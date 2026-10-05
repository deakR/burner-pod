/**
 * websocket.js - WebSocket connection and message handling
 * Manages real-time communication with the chat server.
 */

/**
 * Initializes and returns a WebSocket connection for the given room.
 * Automatically selects ws:// or wss:// based on current page protocol.
 * @param {string} roomName - The room identifier
 * @param {string} username - The user's display name
 * @returns {WebSocket} Connected WebSocket instance
 */
function createWebSocket(roomName, username) {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    return new WebSocket(`${protocol}//${host}/ws/${roomName}?user=${encodeURIComponent(username)}`);
}

/**
 * Sends a message through the WebSocket connection.
 * @param {WebSocket} ws - The WebSocket connection
 * @param {string} username - The sender's username
 * @param {string} text - The message text (can be encrypted)
 */
function sendMessage(ws, username, text) {
    if (ws.readyState === WebSocket.OPEN) {
        const msgObject = {
            username: username,
            text: text
        };
        ws.send(JSON.stringify(msgObject));
    } else {
        alert("Connection lost. Refresh the page.");
    }
}
