use std::{env, fs, io::Read, path::PathBuf};

fn main() {
    println!("cargo:rerun-if-env-changed=KTAV_NATIVE_NOTICES");
    println!("cargo:rerun-if-changed=build.rs");
    let out = PathBuf::from(env::var_os("OUT_DIR").expect("Cargo must provide OUT_DIR"));
    let generated = out.join("native_notices.rs");
    let Some(path) = env::var_os("KTAV_NATIVE_NOTICES") else {
        fs::write(generated, "").expect("write development notice stub");
        return;
    };
    let path = PathBuf::from(path);
    println!("cargo:rerun-if-changed={}", path.display());
    let mut payload = Vec::new();
    fs::File::open(path)
        .expect("open prepared native notice gzip")
        .take(4 * 1024 * 1024 + 1)
        .read_to_end(&mut payload)
        .expect("read prepared native notice gzip");
    assert!(
        payload.starts_with(&[0x1f, 0x8b]) && payload.len() <= 4 * 1024 * 1024,
        "native notice payload must be a bounded gzip"
    );
    let length = u64::try_from(payload.len()).expect("native notice length must fit u64");
    let mut resource = b"KTAV-NOTICES-GZIP-v1\0".to_vec();
    resource.extend_from_slice(&length.to_le_bytes());
    resource.extend_from_slice(&payload);
    fs::write(out.join("native_notices.bin"), resource).expect("write native notice resource");
    fs::write(
        generated,
        r#"/// Статический gzip JSON: маркер, длина LE64 и payload; не передаётся в ktav_free.
#[used]
#[no_mangle]
pub static KTAV_THIRD_PARTY_NOTICES: [u8; include_bytes!(concat!(env!("OUT_DIR"), "/native_notices.bin")).len()] =
    *include_bytes!(concat!(env!("OUT_DIR"), "/native_notices.bin"));
"#,
    )
    .expect("write native notice declaration");
}
