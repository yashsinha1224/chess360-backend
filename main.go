package main

import (
	"example/hello/game"
	"fmt"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	// Create hub instance
	hub := game.NewHub()

	// Serve static files if needed
	router.Static("/static", "./static")

	// WebSocket endpoint - CONNECTED TO YOUR GAME
	router.GET("/ws", game.HandleWebSocket(hub))

	// HTML page
	router.GET("/", func(ctx *gin.Context) {
		ctx.Header("Content-Type", "text/html")
		ctx.String(200, htmlTemplate)
	})

	fmt.Println("Server starting on http://localhost:8080")
	router.Run(":8080")
}

const htmlTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Chess Game</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
            min-height: 100vh;
            display: flex;
            justify-content: center;
            align-items: center;
        }

        .container {
            text-align: center;
            padding: 2rem;
        }

        h1 {
            color: #fff;
            font-size: 3rem;
            margin-bottom: 0.5rem;
            text-shadow: 2px 2px 4px rgba(0,0,0,0.3);
        }

        .subtitle {
            color: #a0a0a0;
            margin-bottom: 3rem;
            font-size: 1.1rem;
        }

        .button-group {
            display: flex;
            gap: 2rem;
            justify-content: center;
            flex-wrap: wrap;
            margin-bottom: 2rem;
        }

        .btn {
            padding: 1rem 2rem;
            font-size: 1.2rem;
            font-weight: 600;
            border: none;
            border-radius: 12px;
            cursor: pointer;
            transition: all 0.3s ease;
            min-width: 200px;
        }

        .btn-online {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            box-shadow: 0 4px 15px rgba(102, 126, 234, 0.4);
        }

        .btn-online:hover {
            transform: translateY(-2px);
            box-shadow: 0 6px 20px rgba(102, 126, 234, 0.5);
        }

        .btn-offline {
            background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
            color: white;
            box-shadow: 0 4px 15px rgba(240, 147, 251, 0.4);
        }

        .btn-offline:hover {
            transform: translateY(-2px);
            box-shadow: 0 6px 20px rgba(240, 147, 251, 0.5);
        }

        .btn:active {
            transform: translateY(0);
        }

        .status-card {
            background: rgba(255, 255, 255, 0.1);
            backdrop-filter: blur(10px);
            border-radius: 16px;
            padding: 1.5rem;
            margin-top: 2rem;
            max-width: 400px;
            margin-left: auto;
            margin-right: auto;
        }

        .status-title {
            color: #a0a0a0;
            font-size: 0.9rem;
            text-transform: uppercase;
            letter-spacing: 2px;
            margin-bottom: 0.5rem;
        }

        .status-message {
            color: #fff;
            font-size: 1.1rem;
            word-break: break-word;
        }

        .status-message.error {
            color: #ff6b6b;
        }

        .status-message.success {
            color: #51cf66;
        }

        .game-board {
            margin-top: 2rem;
            display: none;
        }

        .spinner {
            display: inline-block;
            width: 20px;
            height: 20px;
            border: 2px solid rgba(255,255,255,0.3);
            border-radius: 50%;
            border-top-color: white;
            animation: spin 0.6s linear infinite;
            margin-left: 10px;
        }

        @keyframes spin {
            to { transform: rotate(360deg); }
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>♜ Chess Game</h1>
        <div class="subtitle">Play online with friends or practice offline</div>
        
        <div class="button-group">
            <button class="btn btn-online" id="onlineBtn">
                🌐 Join Online Game
            </button>
            <button class="btn btn-offline" id="offlineBtn">
                🏠 Play Offline
            </button>
        </div>

        <div class="status-card" id="statusCard">
            <div class="status-title">Status</div>
            <div class="status-message" id="statusMessage">Ready to play</div>
        </div>
    </div>

    <script>
        let ws = null;
        let isConnected = false;
        let playerName = null;

        const onlineBtn = document.getElementById('onlineBtn');
        const offlineBtn = document.getElementById('offlineBtn');
        const statusMessage = document.getElementById('statusMessage');

        function updateStatus(msg, isError = false) {
            statusMessage.textContent = msg;
            if (isError) {
                statusMessage.classList.add('error');
                statusMessage.classList.remove('success');
            } else {
                statusMessage.classList.remove('error');
                statusMessage.classList.add('success');
            }
        }

        function addSpinner() {
            if (!statusMessage.querySelector('.spinner')) {
                const spinner = document.createElement('span');
                spinner.className = 'spinner';
                statusMessage.appendChild(spinner);
            }
        }

        function removeSpinner() {
            const spinner = statusMessage.querySelector('.spinner');
            if (spinner) spinner.remove();
        }

        function connectWebSocket() {
            ws = new WebSocket('ws://localhost:8080/ws');
            
            ws.onopen = function() {
                console.log('WebSocket connected');
                isConnected = true;
                updateStatus('Connected! Looking for opponent...');
                addSpinner();
                
                const joinMsg = {
                    type: "join",
                    payload: {
                        name: playerName || "Anonymous"
                    }
                };
                ws.send(JSON.stringify(joinMsg));
            };
            
            ws.onmessage = function(event) {
                console.log('Received:', event.data);
                let data;
                try {
                    data = JSON.parse(event.data);
                } catch(e) {
                    data = { type: "message", payload: event.data };
                }
                
                if (data.type === "game_start") {
                    removeSpinner();
                    updateStatus('Game found! Starting game...');
                    setTimeout(() => {
                        window.location.href = '/game.html';
                    }, 1500);
                } else if (data.type === "waiting") {
                    updateStatus(data.payload.message || 'Waiting for opponent...');
                } else if (data.type === "game_over") {
                    removeSpinner();
                    updateStatus('Game Over! ' + data.payload.winner + ' wins!', false);
                    setTimeout(() => {
                        ws.close();
                        onlineBtn.disabled = false;
                    }, 3000);
                } else if (data.type === "error") {
                    removeSpinner();
                    updateStatus('Error: ' + data.payload.message, true);
                    setTimeout(() => {
                        ws.close();
                        onlineBtn.disabled = false;
                    }, 2000);
                } else {
                    updateStatus('Server: ' + event.data);
                }
            };
            
            ws.onclose = function() {
                console.log('WebSocket disconnected');
                isConnected = false;
                removeSpinner();
                if (onlineBtn) onlineBtn.disabled = false;
                updateStatus('Disconnected from server', true);
                setTimeout(() => {
                    if (statusMessage.textContent.includes('Disconnected')) {
                        updateStatus('Ready to play');
                    }
                }, 3000);
            };
            
            ws.onerror = function(error) {
                console.error('WebSocket error:', error);
                removeSpinner();
                updateStatus('Connection error', true);
                onlineBtn.disabled = false;
            };
        }

        onlineBtn.addEventListener('click', function() {
            if (isConnected) {
                updateStatus('Already connected!');
                return;
            }
            
            onlineBtn.disabled = true;
            playerName = prompt('Enter your name:', 'Player_' + Math.floor(Math.random() * 1000));
            if (!playerName) {
                playerName = 'Anonymous';
            }
            
            updateStatus('Connecting to server...');
            addSpinner();
            connectWebSocket();
        });

        offlineBtn.addEventListener('click', function() {
            updateStatus('Starting offline game...');
            setTimeout(() => {
                alert('Offline mode coming soon! Online mode is ready to test.');
                updateStatus('Ready to play');
            }, 500);
        });
    </script>
</body>
</html>
`
