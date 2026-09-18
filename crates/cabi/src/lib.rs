//! C ABI wrapper around the `ktav` Rust crate, designed for consumption
//! from Go via `purego` (dynamic loading, no cgo on the caller side).
//!
//! ## Wire format
//!
//! Between Go and Rust we exchange **JSON**, not a custom binary. This
//! keeps the FFI boundary tiny and lets each side use
//! its native JSON machinery.
//!
//! Ktav's Integer and Float scalars (spec 0.5, inferred from the scalar
//! body's lexical form — no typed markers) do not map 1:1 onto JSON
//! numbers (JSON cannot represent arbitrary-precision integers). To keep
//! round-trips lossless we use tagged wrappers:
//!
//! - `Value::Integer(s)` ⇄ `{"$i": "<digits>"}`
//! - `Value::Float(s)`   ⇄ `{"$f": "<text>"}`
//!
//! Everything else maps to the obvious JSON shape (`null`, booleans,
//! strings, arrays, objects). Object key order is preserved on both
//! sides (`indexmap` here, `encoding/json` with `json.RawMessage` or an
//! ordered map on the Go side).
//!
//! ## C ABI
//!
//! Eight functions, all use the same "caller-owned pointer, callee-owned
//! buffer" pattern:
//!
//! - `ktav_loads(src, src_len, out_buf, out_len, out_err, out_err_len) -> i32`
//! - `ktav_loads_strict(src, src_len, out_buf, out_len, out_err, out_err_len) -> i32`
//! - `ktav_dumps(src, src_len, out_buf, out_len, out_err, out_err_len) -> i32`
//! - `ktav_dumps_force_strings(src, src_len, out_buf, out_len, out_err, out_err_len) -> i32`
//! - `ktav_emit_canonical(src, src_len, out_buf, out_len, out_err, out_err_len) -> i32`
//! - `ktav_format(src, src_len, out_buf, out_len, out_err, out_err_len) -> i32`
//! - `ktav_free(ptr, len)` — free a buffer returned by any output fn.
//! - `ktav_version()` — NUL-terminated static string, for sanity checks.
//!
//! Return code: `0` on success, `1` on error. On error, `out_err` holds
//! a UTF-8 JSON envelope produced by `ktav::ErrorEnvelope::to_json()` —
//! one JSON object, always nine fields in order (error, reason, line,
//! line_text, span, path, body, canonical, spec_section), with absent
//! info as explicit null — freed via `ktav_free` like any output buffer.

use std::os::raw::{c_char, c_int};
use std::ptr;
use std::slice;

use indexmap::IndexMap;
use ktav::value::{ObjectMap, Value};
use serde::de::{self, MapAccess, Visitor};
use serde::{Deserialize, Deserializer};
use serde_json::{Map as JsonMap, Value as Json};

/// Written into the caller's `**u8` / `*usize` on success.
#[inline]
unsafe fn emit(buf: Vec<u8>, out_buf: *mut *mut u8, out_len: *mut usize) {
    let mut boxed = buf.into_boxed_slice();
    let len = boxed.len();
    let ptr = boxed.as_mut_ptr();
    std::mem::forget(boxed);
    *out_buf = ptr;
    *out_len = len;
}

unsafe fn emit_err(msg: String, out_err: *mut *mut c_char, out_err_len: *mut usize) {
    let bytes = msg.into_bytes();
    let mut boxed = bytes.into_boxed_slice();
    let len = boxed.len();
    let ptr = boxed.as_mut_ptr() as *mut c_char;
    std::mem::forget(boxed);
    *out_err = ptr;
    *out_err_len = len;
}

/// Render the JSON envelope for `err` against the Ktav source text it
/// was (or would have been) parsed from. Written once so every error
/// path leaves the ABI in the same shape.
///
/// `from_error` maps `Error::Message` to an all-null row, dropping the
/// wrapped text; fill `body` with it here so hosts can reconstruct a
/// readable message from the envelope. Every other field keeps the
/// uniform `from_error` semantics.
fn envelope_json(err: &ktav::Error, source: &str) -> String {
    let mut env = ktav::ErrorEnvelope::from_error(err, source);
    if let ktav::Error::Message(text) = err {
        env.body = Some(text.clone());
    }
    env.to_json()
}

/// Parse a Ktav document. Returns JSON bytes on success, error message on
/// failure. Caller frees both via `ktav_free`.
///
/// # Safety
/// `src` must point to `src_len` valid bytes. Output pointers must be
/// valid for writes.
#[no_mangle]
pub unsafe extern "C" fn ktav_loads(
    src: *const u8,
    src_len: usize,
    out_buf: *mut *mut u8,
    out_len: *mut usize,
    out_err: *mut *mut c_char,
    out_err_len: *mut usize,
) -> c_int {
    *out_buf = ptr::null_mut();
    *out_len = 0;
    *out_err = ptr::null_mut();
    *out_err_len = 0;

    let input = match std::str::from_utf8(slice::from_raw_parts(src, src_len)) {
        Ok(s) => s,
        Err(e) => {
            let err = ktav::Error::Message(format!("input is not valid UTF-8: {e}"));
            emit_err(envelope_json(&err, ""), out_err, out_err_len);
            return 1;
        }
    };

    let value = match ktav::parse(input) {
        Ok(v) => v,
        Err(e) => {
            emit_err(envelope_json(&e, input), out_err, out_err_len);
            return 1;
        }
    };

    let json = value_to_json(&value);
    let bytes = match serde_json::to_vec(&json) {
        Ok(b) => b,
        Err(e) => {
            let err = ktav::Error::Message(format!("internal: encode JSON: {e}"));
            emit_err(envelope_json(&err, input), out_err, out_err_len);
            return 1;
        }
    };

    emit(bytes, out_buf, out_len);
    0
}

/// Parse a Ktav document with strict numeric spelling checks. Returns JSON
/// bytes on success, error message on failure. Caller frees both via
/// `ktav_free`.
///
/// # Safety
/// Same as [`ktav_loads`].
#[no_mangle]
pub unsafe extern "C" fn ktav_loads_strict(
    src: *const u8,
    src_len: usize,
    out_buf: *mut *mut u8,
    out_len: *mut usize,
    out_err: *mut *mut c_char,
    out_err_len: *mut usize,
) -> c_int {
    *out_buf = ptr::null_mut();
    *out_len = 0;
    *out_err = ptr::null_mut();
    *out_err_len = 0;

    let input = match std::str::from_utf8(slice::from_raw_parts(src, src_len)) {
        Ok(s) => s,
        Err(e) => {
            let err = ktav::Error::Message(format!("input is not valid UTF-8: {e}"));
            emit_err(envelope_json(&err, ""), out_err, out_err_len);
            return 1;
        }
    };

    let value = match ktav::parse_strict(input) {
        Ok(v) => v,
        Err(e) => {
            emit_err(envelope_json(&e, input), out_err, out_err_len);
            return 1;
        }
    };

    let json = value_to_json(&value);
    let bytes = match serde_json::to_vec(&json) {
        Ok(b) => b,
        Err(e) => {
            let err = ktav::Error::Message(format!("internal: encode JSON: {e}"));
            emit_err(envelope_json(&err, input), out_err, out_err_len);
            return 1;
        }
    };

    emit(bytes, out_buf, out_len);
    0
}

/// Render a JSON document (as produced by `ktav_loads` or built by the
/// caller to the same schema) to Ktav text.
///
/// # Safety
/// Same as [`ktav_loads`].
#[no_mangle]
pub unsafe extern "C" fn ktav_dumps(
    src: *const u8,
    src_len: usize,
    out_buf: *mut *mut u8,
    out_len: *mut usize,
    out_err: *mut *mut c_char,
    out_err_len: *mut usize,
) -> c_int {
    *out_buf = ptr::null_mut();
    *out_len = 0;
    *out_err = ptr::null_mut();
    *out_err_len = 0;

    let bytes = slice::from_raw_parts(src, src_len);
    let wire: WireValue = match serde_json::from_slice(bytes) {
        Ok(w) => w,
        Err(e) => {
            let err = ktav::Error::Message(format!("input JSON: {e}"));
            emit_err(envelope_json(&err, ""), out_err, out_err_len);
            return 1;
        }
    };

    let value = match wire.into_value() {
        Ok(v) => v,
        Err(e) => {
            let err = ktav::Error::Message(e);
            emit_err(envelope_json(&err, ""), out_err, out_err_len);
            return 1;
        }
    };

    if !matches!(value, Value::Object(_) | Value::Array(_)) {
        let err =
            ktav::Error::Message("top-level Ktav document must be an object or array".to_string());
        emit_err(envelope_json(&err, ""), out_err, out_err_len);
        return 1;
    }

    let text = match ktav::render::render(&value) {
        Ok(s) => s,
        Err(e) => {
            emit_err(envelope_json(&e, ""), out_err, out_err_len);
            return 1;
        }
    };

    emit(text.into_bytes(), out_buf, out_len);
    0
}

/// Render a JSON document to Ktav text with **every scalar coerced to
/// a String** — typed integers (`:i`), typed floats (`:f`), booleans,
/// and null are flattened to their textual form via the raw-marker
/// `::`. Compounds preserve their structure. The output round-trips
/// back through `ktav_loads` as the same set of String scalars.
///
/// # Safety
/// Same as [`ktav_loads`].
#[no_mangle]
pub unsafe extern "C" fn ktav_dumps_force_strings(
    src: *const u8,
    src_len: usize,
    out_buf: *mut *mut u8,
    out_len: *mut usize,
    out_err: *mut *mut c_char,
    out_err_len: *mut usize,
) -> c_int {
    *out_buf = ptr::null_mut();
    *out_len = 0;
    *out_err = ptr::null_mut();
    *out_err_len = 0;

    let bytes = slice::from_raw_parts(src, src_len);
    let wire: WireValue = match serde_json::from_slice(bytes) {
        Ok(w) => w,
        Err(e) => {
            let err = ktav::Error::Message(format!("input JSON: {e}"));
            emit_err(envelope_json(&err, ""), out_err, out_err_len);
            return 1;
        }
    };

    let value = match wire.into_value() {
        Ok(v) => v,
        Err(e) => {
            let err = ktav::Error::Message(e);
            emit_err(envelope_json(&err, ""), out_err, out_err_len);
            return 1;
        }
    };

    if !matches!(value, Value::Object(_) | Value::Array(_)) {
        let err =
            ktav::Error::Message("top-level Ktav document must be an object or array".to_string());
        emit_err(envelope_json(&err, ""), out_err, out_err_len);
        return 1;
    }

    let text = match ktav::to_string_force_strings(&value) {
        Ok(s) => s,
        Err(e) => {
            emit_err(envelope_json(&e, ""), out_err, out_err_len);
            return 1;
        }
    };

    emit(text.into_bytes(), out_buf, out_len);
    0
}

/// Render a JSON document (produced by `ktav_loads`) to Ktav text in
/// **canonical form** (spec § 5.9 — byte-deterministic, no inline
/// compounds, canonical float / integer normalisation). Two calls with
/// the same input always produce the same bytes.
///
/// # Safety
/// Same as [`ktav_loads`].
#[no_mangle]
pub unsafe extern "C" fn ktav_emit_canonical(
    src: *const u8,
    src_len: usize,
    out_buf: *mut *mut u8,
    out_len: *mut usize,
    out_err: *mut *mut c_char,
    out_err_len: *mut usize,
) -> c_int {
    *out_buf = ptr::null_mut();
    *out_len = 0;
    *out_err = ptr::null_mut();
    *out_err_len = 0;

    let bytes = slice::from_raw_parts(src, src_len);
    let wire: WireValue = match serde_json::from_slice(bytes) {
        Ok(w) => w,
        Err(e) => {
            let err = ktav::Error::Message(format!("input JSON: {e}"));
            emit_err(envelope_json(&err, ""), out_err, out_err_len);
            return 1;
        }
    };

    let value = match wire.into_value() {
        Ok(v) => v,
        Err(e) => {
            let err = ktav::Error::Message(e);
            emit_err(envelope_json(&err, ""), out_err, out_err_len);
            return 1;
        }
    };

    if !matches!(value, Value::Object(_) | Value::Array(_)) {
        let err =
            ktav::Error::Message("top-level Ktav document must be an object or array".to_string());
        emit_err(envelope_json(&err, ""), out_err, out_err_len);
        return 1;
    }

    let text = match ktav::emit_canonical(&value) {
        Ok(s) => s,
        Err(e) => {
            emit_err(envelope_json(&e, ""), out_err, out_err_len);
            return 1;
        }
    };

    emit(text.into_bytes(), out_buf, out_len);
    0
}

/// Format Ktav **source text** (not JSON — no `WireValue` deserialisation
/// happens here) into its normalised spelling via `ktav::format_str`.
/// Semantics guaranteed upstream:
///
/// - Every comment is preserved verbatim (spec § 3.4: a comment owns a
///   whole line, so attachment is unambiguous).
/// - Blank lines survive as a grouping hint, but a run of 2+ collapses
///   to exactly one and blank padding immediately inside a bracket is
///   dropped — this makes the transform a fixed point
///   (`format(format(x)) == format(x)`).
/// - Key order is never changed (canonical form has no sorting rule,
///   § 5.9).
/// - For a document with no comments **and** no blank lines, the
///   result equals `ktav_emit_canonical` of its parse.
///
/// # Safety
/// Same as [`ktav_loads`].
#[no_mangle]
pub unsafe extern "C" fn ktav_format(
    src: *const u8,
    src_len: usize,
    out_buf: *mut *mut u8,
    out_len: *mut usize,
    out_err: *mut *mut c_char,
    out_err_len: *mut usize,
) -> c_int {
    *out_buf = ptr::null_mut();
    *out_len = 0;
    *out_err = ptr::null_mut();
    *out_err_len = 0;

    let input = match std::str::from_utf8(slice::from_raw_parts(src, src_len)) {
        Ok(s) => s,
        Err(e) => {
            let err = ktav::Error::Message(format!("input is not valid UTF-8: {e}"));
            emit_err(envelope_json(&err, ""), out_err, out_err_len);
            return 1;
        }
    };

    let text = match ktav::format_str(input) {
        Ok(s) => s,
        Err(e) => {
            emit_err(envelope_json(&e, input), out_err, out_err_len);
            return 1;
        }
    };

    emit(text.into_bytes(), out_buf, out_len);
    0
}

/// Free a buffer returned by `ktav_loads` / `ktav_dumps` (success or
/// error). `ptr`/`len` is a no-op when null/zero.
///
/// # Safety
/// Must be called exactly once per returned buffer with the same length
/// it was returned with.
#[no_mangle]
pub unsafe extern "C" fn ktav_free(ptr: *mut u8, len: usize) {
    if ptr.is_null() || len == 0 {
        return;
    }
    let _ = Box::from_raw(std::ptr::slice_from_raw_parts_mut(ptr, len));
}

/// NUL-terminated static version string (crate version). For sanity
/// checks from the Go side that `LoadLibrary` picked up the right file.
#[no_mangle]
pub extern "C" fn ktav_version() -> *const c_char {
    concat!(env!("CARGO_PKG_VERSION"), "\0").as_ptr() as *const c_char
}

// ─── Value ↔ JSON conversion ──────────────────────────────────────────────

fn value_to_json(v: &Value) -> Json {
    match v {
        Value::Null => Json::Null,
        Value::Bool(b) => Json::Bool(*b),
        Value::Integer(s) => {
            let mut m = JsonMap::new();
            m.insert("$i".to_string(), Json::String(s.to_string()));
            Json::Object(m)
        }
        Value::Float(s) => {
            let mut m = JsonMap::new();
            m.insert("$f".to_string(), Json::String(s.to_string()));
            Json::Object(m)
        }
        Value::String(s) => Json::String(s.to_string()),
        Value::Array(a) => Json::Array(a.iter().map(value_to_json).collect()),
        Value::Object(o) => {
            let mut m = JsonMap::new();
            for (k, val) in o {
                m.insert(k.to_string(), value_to_json(val));
            }
            Json::Object(m)
        }
    }
}

/// Deserialize target that understands both plain JSON values and the
/// `{"$i": ...}` / `{"$f": ...}` tagged wrappers, preserving object key
/// order via `indexmap`.
enum WireValue {
    Null,
    Bool(bool),
    Integer(String),
    Float(String),
    String(String),
    Array(Vec<WireValue>),
    Object(IndexMap<String, WireValue>),
}

impl WireValue {
    fn into_value(self) -> Result<Value, String> {
        match self {
            WireValue::Null => Ok(Value::Null),
            WireValue::Bool(b) => Ok(Value::Bool(b)),
            WireValue::Integer(s) => {
                validate_integer(&s)?;
                Ok(Value::Integer(s.into()))
            }
            WireValue::Float(s) => {
                validate_float(&s)?;
                Ok(Value::Float(s.into()))
            }
            WireValue::String(s) => Ok(Value::String(s.into())),
            WireValue::Array(items) => {
                let mut out = Vec::with_capacity(items.len());
                for w in items {
                    out.push(w.into_value()?);
                }
                Ok(Value::Array(out))
            }
            WireValue::Object(m) => {
                let mut obj = ObjectMap::with_capacity_and_hasher(m.len(), Default::default());
                for (k, v) in m {
                    obj.insert(k.into(), v.into_value()?);
                }
                Ok(Value::Object(obj))
            }
        }
    }
}

fn validate_integer(s: &str) -> Result<(), String> {
    let rest = s.strip_prefix('-').unwrap_or(s);
    if rest.is_empty() || !rest.bytes().all(|b| b.is_ascii_digit()) {
        return Err(format!("$i payload not an integer literal: {s:?}"));
    }
    Ok(())
}

fn validate_float(s: &str) -> Result<(), String> {
    if s.parse::<f64>().is_err() {
        return Err(format!("$f payload not a finite decimal: {s:?}"));
    }
    if !s.bytes().any(|b| b == b'.' || b == b'e' || b == b'E') {
        return Err(format!("$f payload must contain '.' or exponent: {s:?}"));
    }
    Ok(())
}

impl<'de> Deserialize<'de> for WireValue {
    fn deserialize<D: Deserializer<'de>>(d: D) -> Result<Self, D::Error> {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = WireValue;
            fn expecting(&self, f: &mut std::fmt::Formatter) -> std::fmt::Result {
                f.write_str("a JSON value")
            }
            fn visit_unit<E: de::Error>(self) -> Result<WireValue, E> {
                Ok(WireValue::Null)
            }
            fn visit_none<E: de::Error>(self) -> Result<WireValue, E> {
                Ok(WireValue::Null)
            }
            fn visit_some<D: Deserializer<'de>>(self, d: D) -> Result<WireValue, D::Error> {
                WireValue::deserialize(d)
            }
            fn visit_bool<E: de::Error>(self, b: bool) -> Result<WireValue, E> {
                Ok(WireValue::Bool(b))
            }
            fn visit_i64<E: de::Error>(self, n: i64) -> Result<WireValue, E> {
                Ok(WireValue::Integer(n.to_string()))
            }
            fn visit_u64<E: de::Error>(self, n: u64) -> Result<WireValue, E> {
                Ok(WireValue::Integer(n.to_string()))
            }
            fn visit_f64<E: de::Error>(self, n: f64) -> Result<WireValue, E> {
                if !n.is_finite() {
                    return Err(E::custom("NaN / ±Infinity not allowed in Ktav"));
                }
                // Bare JSON floats get the ":f" wire form with a forced
                // decimal point so render's grammar check is satisfied.
                let mut s = format!("{n}");
                if !s.contains('.') && !s.contains('e') && !s.contains('E') {
                    s.push_str(".0");
                }
                Ok(WireValue::Float(s))
            }
            fn visit_str<E: de::Error>(self, v: &str) -> Result<WireValue, E> {
                Ok(WireValue::String(v.to_string()))
            }
            fn visit_string<E: de::Error>(self, v: String) -> Result<WireValue, E> {
                Ok(WireValue::String(v))
            }
            fn visit_seq<A: de::SeqAccess<'de>>(self, mut seq: A) -> Result<WireValue, A::Error> {
                let mut out = Vec::new();
                while let Some(item) = seq.next_element()? {
                    out.push(item);
                }
                Ok(WireValue::Array(out))
            }
            fn visit_map<A: MapAccess<'de>>(self, mut map: A) -> Result<WireValue, A::Error> {
                let Some(k1) = map.next_key::<String>()? else {
                    return Ok(WireValue::Object(IndexMap::new()));
                };
                let v1: WireValue = map.next_value()?;
                let second_key: Option<String> = map.next_key()?;

                if second_key.is_none() && (k1 == "$i" || k1 == "$f") {
                    let payload = match v1 {
                        WireValue::String(s) => s,
                        WireValue::Integer(s) => s,
                        WireValue::Float(s) => s,
                        _ => {
                            return Err(de::Error::custom(format!("{k1} payload must be a string")))
                        }
                    };
                    return Ok(if k1 == "$i" {
                        WireValue::Integer(payload)
                    } else {
                        WireValue::Float(payload)
                    });
                }

                let mut out: IndexMap<String, WireValue> = IndexMap::new();
                out.insert(k1, v1);
                if let Some(k2) = second_key {
                    let v2: WireValue = map.next_value()?;
                    out.insert(k2, v2);
                    while let Some((k, v)) = map.next_entry::<String, WireValue>()? {
                        out.insert(k, v);
                    }
                }
                Ok(WireValue::Object(out))
            }
        }

        d.deserialize_any(V)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    const DOC: &str = "\
# top-level comment
key: value

map:
  inner: 1
  # nested comment
  other: two
list:
  - a
  - b
";

    #[test]
    fn format_keeps_comments() {
        let out = ktav::format_str(DOC).unwrap();
        assert!(out.contains("# top-level comment"));
        assert!(out.contains("# nested comment"));
        assert!(out.contains("inner: 1"));
    }

    #[test]
    fn format_is_fixed_point() {
        let once = ktav::format_str(DOC).unwrap();
        let twice = ktav::format_str(&once).unwrap();
        assert_eq!(once, twice);
    }

    #[test]
    fn format_without_comments_or_blanks_equals_canonical() {
        let doc = "a: 1\nb:\n  c: [1, 2, 3]\n";
        let formatted = ktav::format_str(doc).unwrap();
        let canonical = ktav::emit_canonical(&ktav::parse(doc).unwrap()).unwrap();
        assert_eq!(formatted, canonical);
    }

    /// Safety contract: output pointers valid for writes; buffers freed
    /// via `ktav_free`.
    unsafe fn call_format(input: &[u8]) -> (c_int, Option<String>) {
        let mut out_buf: *mut u8 = ptr::null_mut();
        let mut out_len: usize = 0;
        let mut out_err: *mut c_char = ptr::null_mut();
        let mut out_err_len: usize = 0;
        let rc = ktav_format(
            input.as_ptr(),
            input.len(),
            &mut out_buf,
            &mut out_len,
            &mut out_err,
            &mut out_err_len,
        );
        let err = (!out_err.is_null()).then(|| {
            let bytes = slice::from_raw_parts(out_err as *const u8, out_err_len).to_vec();
            ktav_free(out_err as *mut u8, out_err_len);
            String::from_utf8(bytes).unwrap()
        });
        if rc == 0 {
            ktav_free(out_buf, out_len);
        }
        (rc, err)
    }

    #[test]
    fn format_abi_error_envelope_has_ten_fields() {
        // UTF-8 failure path.
        let (rc, err) = unsafe { call_format(b"a: \xff\xfe") };
        assert_eq!(rc, 1);
        let v: Json = serde_json::from_str(&err.unwrap()).unwrap();
        let keys: Vec<&str> = v.as_object().unwrap().keys().map(String::as_str).collect();
        assert_eq!(
            keys,
            [
                "error",
                "reason",
                "line",
                "line_text",
                "span",
                "path",
                "body",
                "canonical",
                "spec_section",
                "message"
            ]
        );
        assert_eq!(v["error"], "Message");
        assert!(v["body"].is_string());
        assert!(v["body"]
            .as_str()
            .unwrap()
            .starts_with("input is not valid UTF-8"));

        // Parse failure path against valid UTF-8 source.
        let (rc, err) = unsafe { call_format(b"a: [") };
        assert_eq!(rc, 1);
        let v: Json = serde_json::from_str(&err.unwrap()).unwrap();
        assert!(v.as_object().unwrap().len() == 10);
        assert!(v["error"].is_string());
    }
}
