/**
 * chat.js - Chat page functionality (chat.html)
 * Handles message display, encryption, countdown timer, and real-time updates.
 */

let ws = null;
let username = "Anonymous";
let roomName = "";
let roomExpiry = 0;
let timerInterval = null;

/**
 * Updates the countdown timer until room self-destructs.
 */
function updateTimer() {
    if (!roomExpiry) return;
    const now = Math.floor(Date.now() / 1000);
    const remaining = roomExpiry - now;
    const timerEl = document.getElementById('room-timer');
    if (!timerEl) return;

    if (remaining <= 0) {
        timerEl.textContent = "⏱️ Expired";
        timerEl.style.color = "#ff4444";
        if (timerInterval) {
            clearInterval(timerInterval);
            timerInterval = null;
        }
        handleDisconnect();
        return;
    }

    const minutes = Math.floor(remaining / 60);
    const seconds = remaining % 60;
    timerEl.textContent = `⏱️ ${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`;
    if (remaining <= 60) {
        timerEl.style.color = "#ff4444";
    } else if (remaining <= 300) {
        timerEl.style.color = "#ffbb00";
    }
}

/**
 * Renders a chat message safely into the message list.
 * @param {Object} data - Message data with username, text, and optional expiresAt properties
 * @param {string} password - Current room key for decryption (optional)
 */
async function renderMessage(data, password = "") {
    const messages = document.getElementById('messages');
    const messageItem = document.createElement('li');

    // Handle system notifications
    if (data.username === "System") {
        messageItem.style.background = "transparent";
        messageItem.style.border = "none";
        const sysDiv = document.createElement('div');
        sysDiv.style.cssText = "text-align: center; color: #666; font-size: 0.65em; font-style: italic; margin: 0; padding: 0; line-height: 1.2;";
        sysDiv.textContent = data.text;
        messageItem.appendChild(sysDiv);
        messages.appendChild(messageItem);
        messages.scrollTop = messages.scrollHeight;
        return;
    }

    messageItem.setAttribute('data-enc', data.text);
    messageItem.setAttribute('data-username', data.username);

    let displayText = data.text;
    if (data.text.startsWith("ENC::")) {
        if (password) {
            try {
                displayText = await decryptMessage(data.text, password, roomName);
            } catch (e) {
                displayText = "🔒 Decryption Failed";
            }
        } else {
            displayText = "🔒 Encrypted Message";
        }
    }

    const wrapper = document.createElement('div');
    wrapper.style.cssText = "margin: 0; padding: 0; line-height: 1.2;";

    const userEl = document.createElement('strong');
    userEl.style.cssText = "display: block; margin: 0; padding: 0; font-size: 0.9em;";
    userEl.textContent = data.username;

    const contentEl = document.createElement('div');
    contentEl.className = "message-text";
    contentEl.textContent = displayText;

    wrapper.appendChild(userEl);
    wrapper.appendChild(contentEl);
    messageItem.appendChild(wrapper);

    messages.appendChild(messageItem);
    messages.scrollTop = messages.scrollHeight;
}

/**
 * Reprocesses all encrypted messages in the chat history.
 * Called when the user enters or changes the room key.
 */
async function reprocessMessages() {
    const keyInput = document.getElementById('room-key').value;
    if (!keyInput) return;

    const messageItems = document.querySelectorAll('#messages li');

    for (let msgItem of messageItems) {
        const rawText = msgItem.getAttribute('data-enc');
        if (rawText && rawText.startsWith("ENC::")) {
            try {
                const decrypted = await decryptMessage(rawText, keyInput, roomName);
                const contentEl = msgItem.querySelector('.message-text');
                if (contentEl) {
                    contentEl.textContent = decrypted;
                }
                msgItem.style.borderLeft = '2px solid #00ff00';
                setTimeout(() => {
                    msgItem.style.borderLeft = '';
                }, 500);
            } catch (e) {
                // Key does not match
            }
        }
    }
}

/**
 * Handles WebSocket disconnection (room expiration or network issues).
 */
function handleDisconnect() {
    const overlay = document.getElementById('expired-overlay');
    if (overlay) overlay.style.display = 'flex';
    const container = document.querySelector('.chat-container');
    if (container) container.style.filter = 'blur(5px)';
    const input = document.getElementById('input');
    if (input) input.disabled = true;
}

/**
 * Handles message form submission.
 */
async function handleMessageSubmit(event) {
    event.preventDefault();
    const input = document.getElementById('input');
    let finalText = input.value;
    if (!finalText.trim()) return;

    const password = document.getElementById('room-key').value;
    if (password) {
        finalText = await encryptMessage(finalText, password, roomName);
    }

    sendMessage(ws, username, finalText);
    input.value = '';
}

/**
 * Sets up 1-click room link sharing with optional zero-knowledge hash.
 */
function initializeShareButton() {
    const shareBtn = document.getElementById('share-btn');
    if (!shareBtn) return;

    shareBtn.addEventListener('click', () => {
        const key = document.getElementById('room-key').value;
        const url = new URL(`/r/${encodeURIComponent(roomName)}`, window.location.origin);
        if (key) {
            url.hash = encodeURIComponent(key);
        }

        if (navigator.clipboard && navigator.clipboard.writeText) {
            navigator.clipboard.writeText(url.href).then(() => {
                const original = shareBtn.textContent;
                shareBtn.textContent = "✓ Copied!";
                setTimeout(() => { shareBtn.textContent = original; }, 1500);
            }).catch(() => {
                prompt("Copy room link:", url.href);
            });
        } else {
            prompt("Copy room link:", url.href);
        }
    });
}

/**
 * Initializes the chat page.
 */
function initializeChat() {
    const urlParams = new URLSearchParams(window.location.search);
    roomName = urlParams.get('room');
    username = urlParams.get('user') || "Anonymous";

    if (!roomName) {
        window.location.href = "/";
        return;
    }

    document.getElementById('room-title').innerText = `Room: ${roomName}`;
    document.title = `Chat - ${roomName}`;

    // Read zero-knowledge secret key from URL hash if provided
    if (window.location.hash && window.location.hash.length > 1) {
        const secret = decodeURIComponent(window.location.hash.substring(1));
        const keyInput = document.getElementById('room-key');
        if (keyInput) keyInput.value = secret;
    }

    window.addEventListener('hashchange', () => {
        if (window.location.hash && window.location.hash.length > 1) {
            document.getElementById('room-key').value = decodeURIComponent(window.location.hash.substring(1));
            reprocessMessages();
        }
    });

    // Establish WebSocket connection
    ws = createWebSocket(roomName, username);

    ws.onopen = function () {
        const messages = document.getElementById('messages');
        messages.scrollTop = messages.scrollHeight;
    };

    ws.onmessage = async function (event) {
        const data = JSON.parse(event.data);
        if (data.expiresAt && !roomExpiry) {
            roomExpiry = data.expiresAt;
            updateTimer();
            if (!timerInterval) {
                timerInterval = setInterval(updateTimer, 1000);
            }
        }
        const password = document.getElementById('room-key').value;
        await renderMessage(data, password);
    };

    ws.onclose = handleDisconnect;

    const form = document.getElementById('form');
    form.addEventListener('submit', handleMessageSubmit);

    const roomKeyInput = document.getElementById('room-key');
    roomKeyInput.addEventListener('input', reprocessMessages);

    initializeShareButton();
}

// Auto-initialize when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initializeChat);
} else {
    initializeChat();
}
