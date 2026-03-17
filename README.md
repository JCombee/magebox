<p align="center">
  <img src="magebox-logo.svg" alt="MageBox" width="400">
</p>

<p align="center">
  <strong>A modern, fast development environment for Magento 2.</strong><br>
  <em>Built for solo developers and teams alike.</em>
</p>

<p align="center">
  <a href="https://github.com/qoliber/magebox/releases"><img src="https://img.shields.io/github/v/release/qoliber/magebox" alt="Release"></a>
  <a href="https://github.com/qoliber/magebox/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License: MIT"></a>
</p>

<p align="center">
  <a href="https://magebox.dev"><strong>Documentation</strong></a> &middot;
  <a href="https://magebox.dev/guide/quick-start">Quick Start</a> &middot;
  <a href="https://magebox.dev/reference/commands">Commands</a> &middot;
  <a href="https://magebox.dev/changelog">Changelog</a>
</p>

---

Native PHP-FPM and Nginx for maximum performance, with Docker only for services like MySQL, Redis, OpenSearch, and Varnish.

## Quick Start

```bash
# Install MageBox
curl -fsSL https://get.magebox.dev | bash

# Set up your environment (one-time)
magebox bootstrap

# Create a new Magento store
magebox new mystore --quick

# Start working
cd mystore && magebox start
```

See the [full installation guide](https://magebox.dev/guide/installation) for platform-specific instructions.

## Why MageBox?

Unlike fully containerized solutions (Warden, DDEV) or generic tools (Laravel Herd), MageBox is purpose-built for Magento:

| | MageBox | Warden / DDEV | Laravel Herd |
|---|---|---|---|
| **PHP execution** | Native (full speed) | Inside container | Native |
| **File I/O** | Native filesystem | Docker bind mounts (slow on macOS) | Native filesystem |
| **Magento-specific** | Built-in commands | Generic | Not Magento-aware |
| **Multi-project** | Simultaneous with auto PHP switching | One at a time (typically) | Simultaneous |
| **Team workflows** | Clone, fetch DB/media, sync | Manual setup | N/A |
| **Services** | Docker (MySQL, Redis, OpenSearch, etc.) | Docker (everything) | Separate installs |

## Architecture

```
┌──────────────────────────────────────────┐
│              Your Machine                │
│                                          │
│  ┌─────────┐  ┌──────────┐              │
│  │  Nginx  │──│ PHP-FPM  │  ← native    │
│  └────┬────┘  └──────────┘              │
│       │                                  │
│  ┌────┴──────────────────────────┐       │
│  │          Docker               │       │
│  │  MySQL  Redis  OpenSearch ... │       │
│  └───────────────────────────────┘       │
└──────────────────────────────────────────┘
```

PHP and Nginx run natively for zero overhead. Stateful services run in Docker for easy management and isolation. One `.magebox.yaml` per project configures everything.

## Features

- **Autostart** - `service install` to start everything on boot — no manual `start` needed
- **Project management** - `init`, `start`, `stop`, `restart`, `status`, `new`
- **PHP control** - Version switching, per-project INI settings, OPcache, isolated PHP-FPM masters
- **Database tools** - Shell, import/export with progress bars, snapshots, process monitor
- **Service logs** - `logs php`, `logs nginx`, `logs mysql`, `logs redis`
- **Debugging** - Xdebug, Blackfire, Tideways integration
- **Varnish** - Enable/disable, VCL management, cache purge
- **SSL** - Automatic certificate generation with local CA
- **DNS** - Automatic via dnsmasq or hosts file
- **Teams** - Clone repos, fetch databases & media from shared storage
- **Expose** - Share local projects via Cloudflare Tunnels
- **Testing** - PHPUnit, PHPStan, PHPCS, PHPMD
- **Multi-domain** - Multiple storefronts with per-domain store codes

For full details, see the [documentation](https://magebox.dev).

## Supported Platforms

| Platform | Status |
|----------|--------|
| macOS (Apple Silicon & Intel) | Fully supported |
| Fedora / RHEL | Fully supported |
| Ubuntu / Debian | Fully supported |
| Arch Linux | Fully supported |

## License

MIT License - see [LICENSE](LICENSE) for details.

Built with care by [Qoliber](https://qoliber.com).
