use actix_web::{get, web, App, HttpResponse, HttpServer};
use enigo::*;
use log::{error, info};
use std::env;
use std::process::{Child, Command};
use std::sync::Arc;
use std::thread;
use std::time::Duration;
use tokio::sync::Mutex;

struct AppState {
    game: Arc<Mutex<Option<Child>>>,
}

#[get("/")]
async fn index() -> HttpResponse {
    HttpResponse::Ok().content_type("text/html").body(
        r#"
        <!DOCTYPE html>
        <html>
        <head>
            <style>
                body, html { margin: 0; padding: 0; height: 100%; overflow: hidden; }
                #gameImage { max-width: 100%; max-height: 100%; object-fit: contain; }
            </style>
        </head>
        <body>
            <img id="gameImage" src="/screenshot" alt="Game Screenshot">
            <script>
                setInterval(() => {
                    document.getElementById('gameImage').src = '/screenshot?' + new Date().getTime();
                }, 1000); // Refresh every second
            </script>
        </body>
        </html>
    "#,
    )
}

#[get("/screenshot")]
async fn screenshot() -> HttpResponse {
    let output = Command::new("ffmpeg")
        .args(&[
            "-f",
            "x11grab",
            "-video_size",
            &env::var("RESOLUTION").unwrap_or_else(|_| "1920x1080".to_string()),
            "-i",
            ":99",
            "-frames:v",
            "1",
            "-f",
            "image2",
            "-",
        ])
        .output()
        .expect("Failed to execute ffmpeg");

    if output.status.success() {
        HttpResponse::Ok()
            .content_type("image/png")
            .body(output.stdout)
    } else {
        let error_message = String::from_utf8_lossy(&output.stderr);
        error!("Failed to capture screenshot: {}", error_message);
        HttpResponse::InternalServerError().body("Failed to capture screenshot")
    }
}

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    env_logger::init_from_env(env_logger::Env::default().default_filter_or("info"));
    info!("Starting server and game");

    let args: Vec<String> = env::args().collect();

    if args.len() != 2 {
        error!("Usage: {} <path_to_gui_application>", args[0]);
        std::process::exit(1);
    }

    let app_path = args[1].clone();

    // Create shared state
    let app_state = web::Data::new(AppState {
        game: Arc::new(Mutex::new(None)),
    });

    // Launch the game in a separate thread
    let game_state = app_state.game.clone();
    thread::spawn(move || {
        let app_dir = std::path::Path::new(&app_path)
            .parent()
            .expect("Failed to get parent directory of application");

        info!("Setting working directory to: {:?}", app_dir);
        if let Err(e) = std::env::set_current_dir(app_dir) {
            error!("Failed to set current directory: {}", e);
            return;
        }

        info!("Launching application: {}", app_path);
        match launch_application(&app_path) {
            Ok(child) => {
                let mut game = game_state.blocking_lock();
                *game = Some(child);
                info!("Game launched successfully");

                if let Err(e) = wait_for_window() {
                    error!("Error waiting for window: {}", e);
                }
                info!("Window detected. Game is running.");
            }
            Err(e) => error!("Failed to launch application: {}", e),
        }
    });

    // Start the web server
    info!("Starting web server on 0.0.0.0:80");
    HttpServer::new(move || {
        App::new()
            .app_data(app_state.clone())
            .service(index)
            .service(screenshot)
    })
    .bind(("0.0.0.0", 80))?
    .run()
    .await
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
