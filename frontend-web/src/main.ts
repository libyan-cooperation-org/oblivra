import './styles/index.css';
import App from './App.svelte';
import { mount } from 'svelte';
import { wsStream } from './lib/stores/websocket.svelte';

// Initialize tactical WebSocket stream
wsStream.connect();

const app = mount(App, { target: document.getElementById('app')! });

export default app;
