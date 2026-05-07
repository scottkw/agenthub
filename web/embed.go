package web

import "embed"

//go:embed dashboard.html terminal.html join.html
//go:embed assets/terminal.js assets/terminal.css
//go:embed assets/dashboard.js assets/dashboard.css
//go:embed assets/join.js assets/join.css
//go:embed vendor/xterm/xterm.js vendor/xterm/xterm.css vendor/xterm/addon-fit.js vendor/xterm/VERSION
//go:embed vendor/xterm/addons/addon-webgl.js vendor/xterm/addons/addon-unicode11.js vendor/xterm/addons/addon-clipboard.js vendor/xterm/addons/addon-search.js vendor/xterm/addons/addon-web-links.js
//go:embed vendor/xterm/addons/addon-image.js
var WebFS embed.FS
