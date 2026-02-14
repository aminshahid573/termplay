# TicTacToe SSH

A real-time multiplayer Tic-Tac-Toe game playable directly in your terminal over SSH. No client installation required—just an SSH client.

## Demo
[Watch the demo video here](YOUR_VIDEO_URL_HERE)

## Features
- **Instant Multiplayer**: Create a room and share the 4-digit code.
- **Public Lobby**: Browse and join open public games.
- **Spectator Mode**: Watch active games in real-time.
- **TUI Interface**: Built with Bubble Tea for a slick terminal UI.
- **Persistence**: Uses Firebase Realtime Database for game state.

## How to Run

1. **Prerequisites**
   - Go 1.25+
   - A Firebase project with Realtime Database enabled.
   - Set `FIREBASE_DB_URL` environment variable (or put `serviceAccount.json` in root).

2. **Start the Server**
   ```bash
   make run
   ```

3. **Connect**
   Open a new terminal window:
   ```bash
   ssh -p 2324 localhost
   ```

## Tech Stack
- **Go**: Backend logic.
- **Bubble Tea**: TUI framework.
- **Wish**: SSH server implementation.
- **Firebase**: Real-time state synchronization.

---
*Created for the love of CLI tools.*
