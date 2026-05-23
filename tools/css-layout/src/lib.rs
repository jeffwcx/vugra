use anyhow::{Context, Result};

pub mod engine;

#[no_mangle]
pub extern "C" fn vuego_css_layout_compute(
    input_ptr: *const u8,
    input_len: usize,
    output_ptr: *mut *mut u8,
    output_len: *mut usize,
) -> i32 {
    match std::panic::catch_unwind(|| compute_ffi(input_ptr, input_len, output_ptr, output_len)) {
        Ok(status) => status,
        Err(_) => write_ffi_output(
            b"panic in vuego_css_layout_compute",
            output_ptr,
            output_len,
            2,
        ),
    }
}

#[no_mangle]
pub extern "C" fn vuego_css_layout_free(ptr: *mut u8, len: usize) {
    if ptr.is_null() || len == 0 {
        return;
    }
    unsafe {
        drop(Box::from_raw(std::slice::from_raw_parts_mut(ptr, len)));
    }
}

fn compute_ffi(
    input_ptr: *const u8,
    input_len: usize,
    output_ptr: *mut *mut u8,
    output_len: *mut usize,
) -> i32 {
    if input_ptr.is_null() || output_ptr.is_null() || output_len.is_null() {
        return write_ffi_output(
            b"null pointer passed to vuego_css_layout_compute",
            output_ptr,
            output_len,
            2,
        );
    }
    let input_bytes = unsafe { std::slice::from_raw_parts(input_ptr, input_len) };
    let result = (|| -> Result<Vec<u8>> {
        let input: engine::Input =
            serde_json::from_slice(input_bytes).context("parse layout input JSON")?;
        let out = engine::compute(input)?;
        serde_json::to_vec(&out).context("serialize layout output JSON")
    })();
    match result {
        Ok(bytes) => write_ffi_output(&bytes, output_ptr, output_len, 0),
        Err(err) => write_ffi_output(err.to_string().as_bytes(), output_ptr, output_len, 1),
    }
}

fn write_ffi_output(
    bytes: &[u8],
    output_ptr: *mut *mut u8,
    output_len: *mut usize,
    status: i32,
) -> i32 {
    if output_ptr.is_null() || output_len.is_null() {
        return 2;
    }
    let mut owned = bytes.to_vec().into_boxed_slice();
    let len = owned.len();
    let ptr = owned.as_mut_ptr();
    std::mem::forget(owned);
    unsafe {
        *output_ptr = ptr;
        *output_len = len;
    }
    status
}
