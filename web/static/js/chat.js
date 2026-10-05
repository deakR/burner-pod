/**
 * chat.js - Chat page functionality (chat.html)
 * Handles message display, encryption, and real-time updates.
 */

let ws = null;
let username = "Anonymous";
let roomName = "";

/**
 * Renders a chat message in the message list.
 * Handles both regular and encrypted messages, applies markdown and sanitization.
 * @param {Object} data - Message data with username and text properties
 * @param {string} password - Current room key for decryption (optional)
 */
async function renderMessage(data, password = "") {
    const messages = document.getElementById('messages');
    const messageItem = document.createElement('li');

    // Handle system messages differently
    if (data.username === "System") {
        messageItem.innerHTML = `<div style="text-align: center; color: #666; font-size: 0.65em; font-style: italic; margin: 0; padding: 0; line-height: 1.2;">${data.text}</div>`;
        messageItem.style.background = "transparent"; 
        messageItem.style.border = "none";
        messages.appendChild(messageItem);
        messages.scrollTop = messages.scrollHeight;
        return;
    }

    // Store the raw ciphertext in a data attribute for later decryption
    messageItem.setAttribute('data-enc', data.text);
    messageItem.setAttribute('data-username', data.username);

    let displayText = data.text;
    if (data.text.startsWith("ENC::")) {
        if (password) {
            try {
                displayText = await decryptMessage(data.text, password, roomName);
            } catch (e) {
                displayText = "🔒 <em>Decryption Failed</em>";
            }
        } else {
            displayText = "🔒 <em>Encrypted Message</em>";
        }
    }

    const rawHtml = marked.parse(displayText);
    const safeHtml = DOMPurify.sanitize(rawHtml);
    const safeUsername = DOMPurify.sanitize(data.username);

    messageItem.innerHTML = `
        <div style="margin: 0; padding: 0; line-height: 1.2;">
            <strong style="display: block; margin: 0; padding: 0; font-size: 0.9em;">${safeUsername}</strong>
            <div style="margin: 0; padding: 0;">${safeHtml}</div>
        </div>
    `;
    
    // Remove any <p> tag margins that marked.js might add
    const paragraphs = messageItem.querySelectorAll('p');
    paragraphs.forEach(p => {
        p.style.margin = '0';
        p.style.padding = '0';
        p.style.display = 'block';
    });
    
    const links = messageItem.querySelectorAll('a');
    links.forEach(link => {
        link.target = '_blank';
        link.style.color = '#00ff00';
    });

    messages.appendChild(messageItem);
    messages.scrollTop = messages.scrollHeight;
}

/**
 * Reprocesses all encrypted messages in the chat history.
 * Called when the user enters or changes the room key.
 * Attempts to decrypt previously locked messages with the current key.
 */
async function reprocessMessages() {
    const keyInput = document.getElementById('room-key').value;
    if (!keyInput) return;

    const messageItems = document.querySelectorAll('#messages li');

    for (let msgItem of messageItems) {
        const rawText = msgItem.getAttribute('data-enc');
        const username = msgItem.getAttribute('data-username');
        
        if (rawText && rawText.startsWith("ENC::")) {
            try {
                const decrypted = await decryptMessage(rawText, keyInput, roomName);
                
                const rawHtml = marked.parse(decrypted);
                const safeHtml = DOMPurify.sanitize(rawHtml);
                const safeUsername = DOMPurify.sanitize(username);
                
                msgItem.innerHTML = `
                    <div style="margin: 0; padding: 0; line-height: 1.2;">
                        <strong style="display: block; margin: 0; padding: 0; font-size: 0.9em;">${safeUsername}</strong>
                        <div style="margin: 0; padding: 0;">${safeHtml}</div>
                    </div>
                `;
                
                const paragraphs = msgItem.querySelectorAll('p');
                paragraphs.forEach(p => {
                    p.style.margin = '0';
                    p.style.padding = '0';
                    p.style.display = 'block';
                });
                
                const links = msgItem.querySelectorAll('a');
                links.forEach(link => {
                    link.target = '_blank';
                    link.style.color = '#00ff00';
                });
                
                // Flash green to show successful decryption
                msgItem.style.borderLeft = '2px solid #00ff00';
                setTimeout(() => {
                    msgItem.style.borderLeft = '';
                }, 500);
                
            } catch (e) {
                console.log("Still can't decrypt this message.");
            }
        }
    }
}

/**
 * Handles WebSocket disconnection (room expiration or network issues).
 * Displays overlay notification and disables the message input.
 */
function handleDisconnect() {
    console.log("Connection closed.");
    document.getElementById('expired-overlay').style.display = 'flex';
    document.querySelector('.chat-container').style.filter = 'blur(5px)';
    document.getElementById('input').disabled = true;
}

/**
 * Handles message form submission.
 * Encrypts the message if a room key is provided, then sends via WebSocket.
 */
async function handleMessageSubmit(event) {
    event.preventDefault();
    const password = document.getElementById('room-key').value;
    const input = document.getElementById('input');
    
    let finalText = input.value;

    if (password) {
        finalText = await encryptMessage(finalText, password, roomName);
    }

    sendMessage(ws, username, finalText);
    input.value = '';
}

/**
 * Initializes the chat page.
 * Extracts room parameters, establishes WebSocket connection, and sets up event handlers.
 */
function initializeChat() {
    // Extract room name and username from URL query parameters
    const urlParams = new URLSearchParams(window.location.search);
    roomName = urlParams.get('room');
    username = urlParams.get('user') || "Anonymous";

    if (!roomName) {
        window.location.href = "/";
        return;
    }

    document.getElementById('room-title').innerText = `Room: ${roomName}`;
    document.title = `Chat - ${roomName}`;

    // Establish WebSocket connection
    ws = createWebSocket(roomName, username);

    ws.onopen = function (event) {
        console.log("Connected to chat.");
        const messages = document.getElementById('messages');
        messages.scrollTop = messages.scrollHeight;
    };

    ws.onmessage = async function (event) {
        const data = JSON.parse(event.data);
        const password = document.getElementById('room-key').value;
        await renderMessage(data, password);
    };

    ws.onclose = handleDisconnect;

    // Set up form submission
    const form = document.getElementById('form');
    form.addEventListener('submit', handleMessageSubmit);

    // Set up room key input handler for reprocessing messages
    const roomKeyInput = document.getElementById('room-key');
    roomKeyInput.addEventListener('input', reprocessMessages);
}

// Auto-initialize when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initializeChat);
} else {
    initializeChat();
}
