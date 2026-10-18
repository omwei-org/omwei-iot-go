use omwei_iot::{decimal_conversion::from_decimal_with_report, decode, encode, SemanticAtom};
use std::io::{self, BufRead};

fn hex(b: &[u8]) -> String {
    b.iter().map(|x| format!("{:02x}", x)).collect()
}

fn main() {
    for line in io::stdin().lock().lines() {
        let line = line.unwrap();
        let mut p = line.splitn(3, '\t');
        let kind = p.next().unwrap();
        let a = p.next().unwrap_or("");
        let b = p.next().unwrap_or("");
        let out = match kind {
            "D" => {
                let id = u16::from_str_radix(a, 16).unwrap();
                match from_decimal_with_report(id, b) {
                    Ok(r) => format!(
                        "OK {} {} {}",
                        r.atom.canonical_value,
                        r.rounded,
                        hex(&encode(r.atom))
                    ),
                    Err(_) => "ERR".to_string(),
                }
            }
            "A" => {
                let id = u16::from_str_radix(a, 16).unwrap();
                let v: i32 = b.parse().unwrap();
                match SemanticAtom::new(id, v) {
                    Ok(x) => format!("OK {}", hex(&encode(x))),
                    Err(_) => "ERR".to_string(),
                }
            }
            "X" => {
                let bytes: Vec<u8> = (0..a.len())
                    .step_by(2)
                    .map(|i| u8::from_str_radix(&a[i..i + 2], 16).unwrap())
                    .collect();
                match decode(&bytes) {
                    Ok(x) => format!("OK {} {}", x.descriptor_id, x.canonical_value),
                    Err(_) => "ERR".to_string(),
                }
            }
            _ => "BAD".to_string(),
        };
        println!("{}", out);
    }
}
