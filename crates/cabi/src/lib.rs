//! Thin C ABI wrapper around the `ktav` crate for consumption from Go
//! via `purego` (dynamic loading, no cgo on the caller side).
//!
//! This crate is a thin shell around [`ktav::declare_cabi!`]: invoking
//! the macro once, at the top level of this cdylib, expands into the
//! whole exported C ABI surface — `ktav_loads`, `ktav_loads_strict`,
//! `ktav_dumps`, `ktav_dumps_force_strings`, `ktav_emit_canonical`,
//! `ktav_canonical_from_source`, `ktav_format`, `ktav_free`,
//! `ktav_version` and `ktav_abi_version` — with the signatures,
//! ownership contract (`ktav_free(ptr, len)` frees exactly what an
//! output function returned) and the JSON error envelope owned
//! upstream by the `ktav` crate. Nothing else belongs here.

ktav::declare_cabi!();
