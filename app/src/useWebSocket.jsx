import { createSignal, onCleanup } from 'solid-js';

var useWebsocket = (url, onData, onError, protocols, reconnectLimit, reconnectInterval) => {
    const [state, setState] = createSignal(WebSocket.CLOSED);

    let socket;
    let reconnectId;
    let reconnections = 0;

    const cancelReconnect = () => {
        if (reconnectId) {
            clearTimeout(reconnectId);
        }
    };

    const connect = () => {
        cancelReconnect();
        setState(WebSocket.CONNECTING);
        socket = new WebSocket(url, protocols);
        socket.onopen = () => setState(WebSocket.OPEN);
        socket.onclose = () => {
            setState(WebSocket.CLOSED);
            if (reconnectLimit && reconnectLimit > reconnections) {
                reconnections += 1;
                reconnectId = setTimeout(connect, reconnectInterval);
            }
        };
        socket.onerror = onError;
        socket.onmessage = onData;
    };

    const disconnect = () => {
        cancelReconnect();
        reconnectLimit = Number.NEGATIVE_INFINITY;
        if (socket) {
            socket.close();
        }
    };

    const changeUrl = (newUrl) => {
        disconnect()
        cancelReconnect();
        setState(WebSocket.CONNECTING);
        socket = new WebSocket(newUrl, protocols);
        socket.onopen = () => setState(WebSocket.OPEN);
        socket.onclose = () => {
            setState(WebSocket.CLOSED);
            if (reconnectLimit && reconnectLimit > reconnections) {
                reconnections += 1;
                reconnectId = setTimeout(connect, reconnectInterval);
            }
        };
        socket.onerror = onError;
        socket.onmessage = onData;
    }

    onCleanup(() => disconnect);
    return [connect, disconnect, changeUrl, state, () => socket];

}

export default useWebsocket;