# 🎸 Kizuna Editor

![License: AGPL v3](https://img.shields.io/badge/License-AGPL_v3-blue.svg)

> **"Don't let your songs die in the voice memo graveyard."**

**Kizuna Editor** is a local-first songwriting environment built for musicians, by a musician. It bridges the chaotic gap between that fleeting melody you recorded on your phone and the final production in your DAW.

It is designed to be the **Safe Harbor** for your creativity: organize lyrics, write tabs, and sync your audio drafts without the bloat of a full studio software.

## 🦉 The Philosophy

I built Kizuna because I was tired of having 500 files named `New Recording 32.m4a` scattered across chats and folders. I wanted a place where I could:
1.  **Capture** an idea instantly.
2.  **Contextualize** it with lyrics and chords.
3.  **Collaborate** with my bandmates without sending zip files back and forth.

This is an **Indie Hacker** project. It's Open Source, privacy-focused, and respects your freedom to self-host.

## ✨ Key Features

- **📝 Distraction-Free Editor:** Write lyrics and chords (AlphaTex) side-by-side with your audio.
- **☁️ Cloud Bridge (Optional):** Sync your "quick riffs" from mobile to desktop instantly.
- **🔒 Local-First:** Your songs live on *your* hard drive. The cloud is just a bridge, not a jail.
- **⚡ Blazing Fast:** Built with Go and Wails. No heavy electron bloat.

## 🛠 Tech Stack

- **Core:** Go (Golang) 1.22+
- **Desktop Engine:** [Wails](https://wails.io/) (Native performance + Web Frontend)
- **UI/UX:** Svelte + TypeScript
- **Persistence:** SQLite (Robust local storage)
- **Notation:** AlphaTab (Rendering music from text)

## 📂 Project Structure (Monorepo)

- `/client`: The Desktop Application. Where the magic happens.
- `/server`: The Kizuna Cloud API. Handles the sync (AGPLv3).
- `/shared`: The Source of Truth. Shared logic between client and server.

## 🚀 Getting Started

### Prerequisites
- Go 1.22+
- Node.js & npm

### Installation

1. Clone the repo:
   ```bash
   git clone [https://github.com/Chinano9/kizuna-editor.git](https://github.com/Chinano9/kizuna-editor.git)
   cd kizuna-editor
   ```

2. Sync the workspace:
   ```bash
   go work sync
   ```

3. Run the Desktop App:
   ```bash
   cd client
   wails dev
   ```

## 🗺️ Roadmap

- [x] Core Architecture (Monorepo)
- [x] Local Database & Editor
- [ ] Cloud Sync Implementation
- [ ] **Mobile Companion PWA** (The "Idea Catcher")
- [ ] Band Collaboration Mode

## 🤝 Contributing

This project is **Open Source** because music belongs to everyone.
Feel free to open issues, suggest features, or submit PRs. Let's build the ultimate songwriter's tool together.

## 📝 License

This project is licensed under the **GNU Affero General Public License v3.0 (AGPL-3.0)**.
See the [LICENSE](LICENSE) file for details.

*In short: You are free to use, modify, and distribute this software, but you must disclose the source code of any modifications, especially if you run it as a service over a network.*

---
*Created with ❤️ and ☕ by Fernando Ponce Solis (@Chinano9)* 🦉
