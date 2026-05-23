use anyhow::{Context, Result};
use std::io::{self, Read};
use vuego_css_layout::engine::{compute, Input};

fn main() -> Result<()> {
    let mut raw = String::new();
    io::stdin()
        .read_to_string(&mut raw)
        .context("read layout input JSON")?;
    let input: Input = serde_json::from_str(&raw).context("parse layout input JSON")?;
    let out = compute(input)?;
    println!("{}", serde_json::to_string_pretty(&out)?);
    Ok(())
}
