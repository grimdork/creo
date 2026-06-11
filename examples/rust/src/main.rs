use std::io::Write;
use std::net::{TcpListener, TcpStream};

fn handle(mut stream: TcpStream) {
    let resp = b"HTTP/1.0 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 16\r\nConnection: close\r\n\r\nhello from Rust\n";
    let _ = stream.write_all(resp);
}

fn main() -> std::io::Result<()> {
    let listener = TcpListener::bind("0.0.0.0:8080")?;
    for stream in listener.incoming() {
        match stream {
            Ok(s) => handle(s),
            Err(e) => eprintln!("accept: {e}"),
        }
    }
    Ok(())
}
