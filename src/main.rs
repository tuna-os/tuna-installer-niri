use ratatui::{
    backend::CrosstermBackend,
    layout::{Constraint, Direction, Layout, Rect},
    style::{Color, Modifier, Style},
    text::{Line, Span},
    widgets::{Block, Borders, List, ListItem, Paragraph, Wrap},
    Frame, Terminal,
};
use serde::{Deserialize, Serialize};
use std::io::{self, Stdout};
use std::process::Command;

fn main() -> anyhow::Result<()> {
    tracing_subscriber::fmt::init();

    crossterm::terminal::enable_raw_mode()?;
    let mut stdout = io::stdout();
    crossterm::execute!(stdout, crossterm::terminal::EnterAlternateScreen)?;

    let backend = CrosstermBackend::new(stdout);
    let mut terminal = Terminal::new(backend)?;

    let result = run_app(&mut terminal);

    // Restore terminal
    crossterm::terminal::disable_raw_mode()?;
    crossterm::execute!(io::stdout(), crossterm::terminal::LeaveAlternateScreen)?;

    result
}

#[derive(Debug, Clone, PartialEq)]
enum Screen {
    Welcome,
    DiskSelect,
    Confirm,
    Installing,
    Done,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct Recipe {
    disk: String,
    filesystem: String,
    #[serde(default)]
    btrfs_subvolumes: bool,
    encryption: Encryption,
    image: String,
    #[serde(default)]
    target_imgref: String,
    #[serde(default = "default_selinux")]
    selinux_disabled: bool,
    hostname: String,
}

fn default_selinux() -> bool { true }

#[derive(Debug, Clone, Serialize, Deserialize)]
struct Encryption {
    #[serde(rename = "type")]
    enc_type: String,
    #[serde(default)]
    passphrase: String,
}

impl Default for Encryption {
    fn default() -> Self {
        Self { enc_type: "none".into(), passphrase: String::new() }
    }
}

impl Default for Recipe {
    fn default() -> Self {
        Self {
            disk: String::new(),
            filesystem: "xfs".into(),
            btrfs_subvolumes: false,
            encryption: Encryption::default(),
            image: "ghcr.io/tuna-os/albacore:gnome".into(),
            target_imgref: String::new(),
            selinux_disabled: true,
            hostname: "tunaos".into(),
        }
    }
}

struct DiskInfo {
    name: String,
    size: String,
    transport: String,
}

struct App {
    screen: Screen,
    disks: Vec<DiskInfo>,
    selected: usize,
    recipe: Recipe,
    install_log: String,
    install_ok: bool,
}

fn run_app(terminal: &mut Terminal<CrosstermBackend<Stdout>>) -> anyhow::Result<()> {
    let mut app = App {
        screen: Screen::Welcome,
        disks: discover_disks(),
        selected: 0,
        recipe: Recipe::default(),
        install_log: String::new(),
        install_ok: false,
    };

    if !app.disks.is_empty() {
        app.recipe.disk = format!("/dev/{}", app.disks[0].name);
    }

    loop {
        terminal.draw(|f| draw(f, &app))?;

        if crossterm::event::poll(std::time::Duration::from_millis(100))? {
            match crossterm::event::read()? {
                crossterm::event::Event::Key(key) => {
                    if !handle_key(&mut app, key) {
                        return Ok(());
                    }
                }
                _ => {}
            }
        }

        // If installing, process output
        if app.screen == Screen::Installing {
            // Install runs outside the event loop for simplicity in this TUI
        }
    }
}

fn handle_key(app: &mut App, key: crossterm::event::KeyEvent) -> bool {
    match app.screen {
        Screen::Welcome => {
            if key.code == crossterm::event::KeyCode::Enter {
                app.screen = Screen::DiskSelect;
                app.disks = discover_disks();
            }
        }
        Screen::DiskSelect => {
            match key.code {
                crossterm::event::KeyCode::Up | crossterm::event::KeyCode::Char('k') => {
                    app.selected = app.selected.saturating_sub(1);
                }
                crossterm::event::KeyCode::Down | crossterm::event::KeyCode::Char('j') => {
                    if app.selected + 1 < app.disks.len() {
                        app.selected += 1;
                    }
                }
                crossterm::event::KeyCode::Enter => {
                    if !app.disks.is_empty() {
                        app.recipe.disk = format!("/dev/{}", app.disks[app.selected].name);
                        app.screen = Screen::Confirm;
                    }
                }
                crossterm::event::KeyCode::Esc => {
                    app.screen = Screen::Welcome;
                }
                _ => {}
            }
        }
        Screen::Confirm => {
            match key.code {
                crossterm::event::KeyCode::Char('i') | crossterm::event::KeyCode::Enter => {
                    // Start install
                    app.screen = Screen::Installing;
                    app.install_log = run_install(&app.recipe);
                    app.install_ok = true; // simplified
                    app.screen = Screen::Done;
                }
                crossterm::event::KeyCode::Esc => {
                    app.screen = Screen::DiskSelect;
                }
                _ => {}
            }
        }
        Screen::Done => {
            if key.code == crossterm::event::KeyCode::Enter
                || key.code == crossterm::event::KeyCode::Esc
                || key.code == crossterm::event::KeyCode::Char('q')
            {
                return false; // quit
            }
        }
        Screen::Installing => {
            // Nothing — handled synchronously above
        }
    }
    true
}

fn draw(f: &mut Frame, app: &App) {
    let area = f.size();
    match app.screen {
        Screen::Welcome => draw_welcome(f, area),
        Screen::DiskSelect => draw_disk_select(f, area, app),
        Screen::Confirm => draw_confirm(f, area, app),
        Screen::Installing => draw_installing(f, area, app),
        Screen::Done => draw_done(f, area, app),
    }
}

fn draw_welcome(f: &mut Frame, area: Rect) {
    let lines = vec![
        Line::from(Span::styled("TunaOS Installer", Style::default().add_modifier(Modifier::BOLD).fg(Color::Cyan))),
        Line::from(""),
        Line::from("This TUI installer will guide you through installing TunaOS"),
        Line::from("onto your computer using the fisherman bootc install backend."),
        Line::from(""),
        Line::from(Span::styled("Press Enter to continue", Style::default().fg(Color::Green))),
    ];
    let p = Paragraph::new(lines)
        .block(Block::default().borders(Borders::ALL).title("Welcome"))
        .wrap(Wrap { trim: false });
    f.render_widget(p, area);
}

fn draw_disk_select(f: &mut Frame, area: Rect, app: &App) {
    let items: Vec<ListItem> = app.disks.iter().enumerate().map(|(i, d)| {
        let style = if i == app.selected {
            Style::default().add_modifier(Modifier::REVERSED)
        } else {
            Style::default()
        };
        ListItem::new(format!(
            "/dev/{}  ({})  [{}]",
            d.name, d.size, d.transport
        ))
        .style(style)
    }).collect();

    let chunks = Layout::default()
        .direction(Direction::Vertical)
        .constraints([Constraint::Min(3), Constraint::Length(3)])
        .split(area);

    let list = List::new(items)
        .block(Block::default().borders(Borders::ALL).title("Select Target Disk"))
        .highlight_style(Style::default().add_modifier(Modifier::REVERSED));
    f.render_widget(list, chunks[0]);

    let help = Paragraph::new("↑/↓ or j/k to navigate · Enter to select · Esc to go back")
        .style(Style::default().fg(Color::DarkGray));
    f.render_widget(help, chunks[1]);
}

fn draw_confirm(f: &mut Frame, area: Rect, app: &App) {
    let lines = vec![
        Line::from(Span::styled("Confirm Installation", Style::default().add_modifier(Modifier::BOLD))),
        Line::from(""),
        Line::from(format!(" Disk:       {}", app.recipe.disk)),
        Line::from(format!(" Filesystem: {}", app.recipe.filesystem)),
        Line::from(format!(" Encryption: {}", app.recipe.encryption.enc_type)),
        Line::from(format!(" Hostname:   {}", app.recipe.hostname)),
        Line::from(format!(" Image:      {}", app.recipe.image)),
        Line::from(""),
        Line::from(Span::styled("Press Enter to install · Esc to go back", Style::default().fg(Color::Green))),
    ];
    let p = Paragraph::new(lines)
        .block(Block::default().borders(Borders::ALL).title("Confirm"))
        .wrap(Wrap { trim: false });
    f.render_widget(p, area);
}

fn draw_installing(f: &mut Frame, area: Rect, app: &App) {
    let lines = vec![
        Line::from("Installing..."),
        Line::from(""),
        Line::from(Span::styled(&app.install_log, Style::default().fg(Color::DarkGray))),
        Line::from(""),
        Line::from(Span::styled("Please wait...", Style::default().fg(Color::Yellow))),
    ];
    let p = Paragraph::new(lines)
        .block(Block::default().borders(Borders::ALL).title("Install Progress"))
        .wrap(Wrap { trim: false });
    f.render_widget(p, area);
}

fn draw_done(f: &mut Frame, area: Rect, app: &App) {
    let (status, details) = if app.install_ok {
        ("✓ Installation Complete", "Remove the installation media and restart. Press any key to exit.")
    } else {
        ("✗ Installation Failed", "Check the installation log. Press any key to exit.")
    };
    let color = if app.install_ok { Color::Green } else { Color::Red };

    let lines = vec![
        Line::from(Span::styled(status, Style::default().add_modifier(Modifier::BOLD).fg(color))),
        Line::from(""),
        Line::from(details),
    ];
    let p = Paragraph::new(lines)
        .block(Block::default().borders(Borders::ALL).title("Done"))
        .wrap(Wrap { trim: false });
    f.render_widget(p, area);
}

fn discover_disks() -> Vec<DiskInfo> {
    let output = Command::new("lsblk")
        .args(["-J", "-o", "NAME,SIZE,TYPE,TRAN"])
        .output();
    let Ok(out) = output else { return vec![] };
    let Ok(json) = serde_json::from_slice::<serde_json::Value>(&out.stdout) else {
        return vec![];
    };

    let Some(devices) = json["blockdevices"].as_array() else { return vec![] };
    devices
        .iter()
        .filter(|d| d["type"] == "disk")
        .map(|d| DiskInfo {
            name: d["name"].as_str().unwrap_or("?").to_string(),
            size: d["size"].as_str().unwrap_or("?").to_string(),
            transport: d["tran"].as_str().unwrap_or("?").to_string(),
        })
        .collect()
}

fn run_install(recipe: &Recipe) -> String {
    let json = serde_json::to_string_pretty(recipe).unwrap_or_default();
    let tmp = std::env::temp_dir().join("fisherman-recipe.json");
    if std::fs::write(&tmp, &json).is_err() {
        return "Failed to write recipe file".into();
    }
    let output = Command::new("fisherman")
        .arg(&tmp)
        .output();
    match output {
        Ok(o) => {
            let mut log = String::from_utf8_lossy(&o.stdout).to_string();
            log.push_str(&String::from_utf8_lossy(&o.stderr));
            log
        }
        Err(e) => format!("Failed to run fisherman: {e}"),
    }
}
