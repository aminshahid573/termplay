# TicTacToe SSH

A real-time multiplayer Tic-Tac-Toe and Chess game playable directly in your terminal over SSH. No client installation required, just an SSH client.

## Demo
https://github.com/user-attachments/assets/0bfcccc5-ecb1-45cc-b4f4-55dc0bf07b33



## Features
- **Two Games in One**: Play Tic-Tac-Toe or Chess.
- **Instant Multiplayer**: Create a room and share the 4-digit code.
- **Public Lobby**: Browse and join open public games.
- **Spectator Mode**: Watch active games in real-time.
- **TUI Interface**: Built with Bubble Tea for a slick terminal UI.
- **Persistence**: Uses Firebase Realtime Database for game state.

## How to Run

### Play Now (Public Server)
> **Note:** The public server is currently under deployment. Check back soon!

Once deployed, you can play instantly:
```bash
ssh tictactoe.example.com
```

### Development (Local)
1. **Get Credentials**:
   - Go to Firebase Console > Project Settings > Service Accounts.
   - Click "Generate New Private Key". This downloads a JSON file.
   - Rename it to `serviceAccount.json` and place it in the project root.
   - **Important**: This file contains secrets! Never commit it. (It is already `.gitignore`d).

2. **Configure Environment**:
   Create a `.env` file in the project root:
   ```env
   FIREBASE_DB_URL=https://YOUR-PROJECT-ID-default-rtdb.firebaseio.com
   GOOGLE_APPLICATION_CREDENTIALS=./serviceAccount.json
   ```
   *(Alternatively, export them as system environment variables)*.

3. **Build**
   ```bash
   make build
   ```

4. **Run**:
   ```bash
   make run
   ```

5. **Connect**:
   ```bash
   ssh -p 2324 localhost
   ```

### Docker

1. **Build**:
   ```bash
   docker build -t tictactoe-ssh .
   ```

2. **Run**:
   Mount your `serviceAccount.json` and set the env var:
   ```bash
   docker run -d -p 2324:2324 \
     -e FIREBASE_DB_URL="https://YOUR-PROJECT-ID-default-rtdb.firebaseio.com" \
     -e GOOGLE_APPLICATION_CREDENTIALS="/app/serviceAccount.json" \
     -v "$(pwd)/serviceAccount.json:/app/serviceAccount.json" \
     --name tictactoe \
     tictactoe-ssh
   ```

## Tech Stack
- **Go**: Backend logic.
- **Bubble Tea**: TUI framework.
- **Wish**: SSH server implementation.
- **Firebase**: Real-time state synchronization.

---
*Created for the love of CLI tools.*
