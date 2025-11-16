use crate::constants::{GREEN, RESET};
use crate::version::VERSION;

#[allow(dead_code)]
pub fn ascii_logo() -> String {
    format!(
        r#"
██████╗ ███████╗ ██████╗ █████╗ ██╗     ██╗     ███████╗██████╗
██╔══██╗██╔════╝██╔════╝██╔══██╗██║     ██║     ██╔════╝██╔══██╗
██████╔╝█████╗  ██║     ███████║██║     ██║     █████╗  ██████╔╝
██╔══██╗██╔══╝  ██║     ██╔══██║██║     ██║     ██╔══╝  ██╔══██╗
██║  ██║███████╗╚██████╗██║  ██║███████╗███████╗███████╗██║  ██║
╚═╝  ╚═╝╚══════╝ ╚═════╝╚═╝  ╚═╝╚══════╝╚══════╝╚══════╝╚═╝  ╚═╝
Blazing-fast command history search with instant documentation and terminal execution [Version: {green}{version}{reset}]

Copyright @ Naren Yellavula (Please give us a star ⭐ here: https://github.com/cybrota/recaller)

"#,
        green = GREEN,
        version = VERSION,
        reset = RESET,
    )
}
