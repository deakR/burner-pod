/**
 * lobby.js - Lobby page functionality (index.html)
 * Handles room creation, joining, and public room list display.
 */

/**
 * Calculates the total time-to-live in seconds from the duration inputs.
 * Multiplies the number input by the selected unit multiplier and updates the hidden TTL field.
 */
function updateTTL() {
    try {
        const number = document.getElementById('duration-number').value;
        const multiplier = document.getElementById('duration-unit').value;
        document.getElementById('ttl-final').value = number * multiplier;
    } catch (e) {
        console.error("TTL Calc failed", e);
    }
}

/**
 * Re-enables the create button when user navigates back using browser history.
 * Prevents the button from staying disabled after page restoration from cache.
 */
function resetCreateButton() {
    const btn = document.querySelector('#create-form button[type="submit"]');
    if (btn) {
        btn.disabled = false;
        btn.innerText = "Create New Room";
    }
}

/**
 * Handles room creation form submission.
 * Prevents default form action, gathers user input, disables submit button,
 * and submits the form programmatically with all required data.
 */
function initializeCreateForm() {
    const createForm = document.getElementById('create-form');
    if (!createForm) return;

    createForm.addEventListener('submit', function(event) {
        event.preventDefault(); 
        
        const btn = createForm.querySelector('button[type="submit"]');
        
        // Prepare Data
        updateTTL(); 
        const name = document.getElementById('username').value;
        if (name) document.getElementById('create-user').value = name;
        
        const isPublic = document.getElementById('public-check').checked;
        document.getElementById('public-final').value = isPublic ? "true" : "false";

        // Disable button and Submit
        if (btn) {
            btn.disabled = true;
            btn.innerText = "Creating...";
        }
        
        // Force submit safely
        createForm.submit();
    });
}

/**
 * Handles room join form submission.
 * Redirects user to the chat page with the specified room code and username.
 */
function initializeJoinForm() {
    const joinForm = document.getElementById('join-form');
    if (!joinForm) return;

    joinForm.addEventListener('submit', function (event) {
        event.preventDefault();
        const room = document.getElementById('room-input').value;
        let user = document.getElementById('username').value;
        if (!user) user = "Anonymous";
        
        window.location.href = `/chat?room=${encodeURIComponent(room)}&user=${encodeURIComponent(user)}`;
    });
}

/**
 * Creates a clickable room chip element for the public rooms list.
 * @param {Object} room - Room data with id and count properties
 * @returns {HTMLElement} The room chip element
 */
function createRoomChip(room) {
    const chip = document.createElement('div');
    chip.className = 'room-chip'; 
    chip.style.cssText = 'border: 1px solid #333; padding: 8px 12px; cursor: pointer; transition: all 0.2s; background: #0a0a0a; margin-right: 5px; margin-bottom: 5px; display: inline-block;';

    chip.innerHTML = `
        <span style="color: #fff; font-weight: bold;">${room.id}</span>
        <span style="color: #00ff00; margin-left: 10px; font-size: 0.8em;">●</span>
        <span style="color: #888; font-size: 0.8em;"> ${room.count}</span>
    `;

    chip.onclick = () => {
        let user = document.getElementById('username').value;
        if (!user) user = "Anonymous";
        window.location.href = `/chat?room=${encodeURIComponent(room.id)}&user=${encodeURIComponent(user)}`;
    };

    chip.onmouseover = () => { chip.style.borderColor = '#fff'; };
    chip.onmouseout = () => { chip.style.borderColor = '#333'; };

    return chip;
}

/**
 * Fetches and displays the list of public rooms from the server.
 * Creates clickable room chips showing room ID and current participant count.
 * Automatically refreshes every 5 seconds to show live room status.
 */
function loadActiveRooms() {
    fetch('/api/rooms')
        .then(response => response.json())
        .then(rooms => {
            const list = document.getElementById('active-rooms-list');
            if (!list) return;
            list.innerHTML = '';

            if (rooms.length === 0) {
                list.innerHTML = '<p style="color: #444; font-size: 0.9rem;">NO_ACTIVE_SIGNALS</p>';
                return;
            }

            rooms.forEach(room => {
                list.appendChild(createRoomChip(room));
            });
        })
        .catch(err => console.error("Signal lost:", err));
}

/**
 * Initializes the lobby page.
 * Sets up form handlers, loads public rooms, and starts auto-refresh.
 */
function initializeLobby() {
    // Reset create button on page show (browser back navigation)
    window.addEventListener('pageshow', resetCreateButton);
    
    // Initialize forms
    initializeCreateForm();
    initializeJoinForm();
    
    // Add event listeners for TTL calculation
    const durationNumber = document.getElementById('duration-number');
    const durationUnit = document.getElementById('duration-unit');
    if (durationNumber) durationNumber.addEventListener('input', updateTTL);
    if (durationUnit) durationUnit.addEventListener('change', updateTTL);
    
    // Initialize TTL and load rooms
    updateTTL();
    loadActiveRooms();
    setInterval(loadActiveRooms, 5000);
}

// Auto-initialize when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initializeLobby);
} else {
    initializeLobby();
}
