/**
 * crypto.js - Client-side AES-GCM encryption/decryption utilities
 * Provides end-to-end encryption for chat messages using password-derived keys.
 */

// In-memory keyCache mapping `${roomId}:${password}` -> CryptoKey to prevent PBKDF2 re-derivation lag
const keyCache = new Map();

/**
 * Derives an AES-GCM encryption key from a password using PBKDF2.
 * @param {string} password - The user's password/room key
 * @param {string} [roomId=""] - The room ID for salt derivation
 * @returns {Promise<CryptoKey>} A 256-bit AES-GCM key for encryption/decryption
 */
async function getKey(password, roomId = "") {
    const cacheKey = `${roomId}:${password}`;
    if (keyCache.has(cacheKey)) {
        return keyCache.get(cacheKey);
    }

    const enc = new TextEncoder();
    const saltStr = roomId ? `burner-pod:${roomId}` : "burner-pod-default-salt";
    const keyMaterial = await window.crypto.subtle.importKey(
        "raw", enc.encode(password), { name: "PBKDF2" }, false, ["deriveKey"]
    );
    const key = await window.crypto.subtle.deriveKey(
        { name: "PBKDF2", salt: enc.encode(saltStr), iterations: 100000, hash: "SHA-256" },
        keyMaterial, { name: "AES-GCM", length: 256 }, false, ["encrypt", "decrypt"]
    );

    keyCache.set(cacheKey, key);
    return key;
}

/**
 * Encrypts a text message using AES-GCM with a password-derived key.
 * @param {string} text - The plain text message to encrypt
 * @param {string} password - The encryption password/room key
 * @param {string} [roomId=""] - The room ID for salt derivation
 * @returns {Promise<string>} Encrypted message prefixed with "ENC::" or original text if encryption fails
 */
async function encryptMessage(text, password, roomId = "") {
    try {
        const key = await getKey(password, roomId);
        const iv = window.crypto.getRandomValues(new Uint8Array(12));
        const encoded = new TextEncoder().encode(text);
        
        const encrypted = await window.crypto.subtle.encrypt(
            { name: "AES-GCM", iv: iv }, key, encoded
        );

        const ivArr = Array.from(iv);
        const encArr = Array.from(new Uint8Array(encrypted));
        const payload = { iv: ivArr, data: encArr };
        return "ENC::" + JSON.stringify(payload);
    } catch (e) {
        console.error("Encryption failed:", e);
        return text;
    }
}

/**
 * Decrypts an encrypted message using the provided password.
 * @param {string} cipherText - The encrypted message (should start with "ENC::")
 * @param {string} password - The decryption password/room key
 * @param {string} [roomId=""] - The room ID for salt derivation
 * @returns {Promise<string>} Decrypted text, or locked message indicator if password is wrong
 */
async function decryptMessage(cipherText, password, roomId = "") {
    if (!cipherText.startsWith("ENC::")) return cipherText;

    try {
        const raw = JSON.parse(cipherText.substring(5));
        const key = await getKey(password, roomId);
        const iv = new Uint8Array(raw.iv);
        const data = new Uint8Array(raw.data);

        const decrypted = await window.crypto.subtle.decrypt(
            { name: "AES-GCM", iv: iv }, key, data
        );

        return new TextDecoder().decode(decrypted);
    } catch (e) {
        return "🔒 <em>Encrypted Message (Wrong Key)</em>";
    }
}
