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

### Development (Local)
1. **Get Credentials**:
   - Go to Firebase Console > Project Settings > Service Accounts.
   - Click "Generate New Private Key". This downloads a JSON file.
   - Rename it to `serviceAccount.json` and place it in the project root.
   - **Important**: This file contains secrets! Never commit it. (It is already `.gitignore`d).

2. **Configure Environment**:
   ```bash
   # Replace with your actual DB URL (from Firebase Console > Realtime Database)
   export FIREBASE_DB_URL="https://YOUR-PROJECT-ID-default-rtdb.firebaseio.com"
   export GOOGLE_APPLICATION_CREDENTIALS="./serviceAccount.json"
   ```

3. **Run**:
   ```bash
   make run
   ```

4. **Connect**:
   ```bash
   ssh -p 2324 localhost
   ```

### Production (Deploy)
1. Build the binary:
   ```bash
   go build -o server ./cmd/server
   ```
2. Set the environment variables (`FIREBASE_DB_URL` and `GOOGLE_APPLICATION_CREDENTIALS`) on your server.
3. Run `./server`.

## Tech Stack
- **Go**: Backend logic.
- **Bubble Tea**: TUI framework.
- **Wish**: SSH server implementation.
- **Firebase**: Real-time state synchronization.

---
*Created for the love of CLI tools.*
