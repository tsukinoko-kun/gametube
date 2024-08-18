use enigo::*;
use std::env;
use std::process::{Child, Command};
use std::thread;
use std::time::Duration;

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let args: Vec<String> = env::args().collect();

    if args.len() != 2 {
        eprintln!("Usage: {} <path_to_gui_application>", args[0]);
        std::process::exit(1);
    }

    let app_path = &args[1];

    println!("Launching application: {}", app_path);

    let mut child = launch_application(app_path)?;

    println!("Waiting for window to spawn...");
    wait_for_window()?;

    println!("Window detected. Exiting.");

    // Optionally, you can close the child process here
    child.kill()?;

    Ok(())
}

fn launch_application(path: &str) -> Result<Child, std::io::Error> {
    Command::new(path).spawn()
}

fn wait_for_window() -> Result<(), Box<dyn std::error::Error>> {
    let enigo = Enigo::new(&Settings {
        ..Default::default()
    })?;

    for _ in 0..60 {
        // Try for 60 seconds
        if let Ok((_, _)) = enigo.location() {
            return Ok(());
        }
        thread::sleep(Duration::from_secs(1));
    }

    Err("Timeout: Window did not spawn within 60 seconds".into())
}
