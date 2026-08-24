import { useCallback, useEffect, useRef, useState } from "react";
import { Send } from "lucide-react";
import { bookingSocketURL, endpoints, errorMessage } from "../api/client";
import { useAuth } from "../context/AuthContext.jsx";
import { useToast } from "../context/ToastContext.jsx";

export default function ChatWindow({ bookingId }) {
  const { user } = useAuth();
  const toast = useToast();
  const [messages, setMessages] = useState([]);
  const [content, setContent] = useState("");
  const [connected, setConnected] = useState(false);
  const [sending, setSending] = useState(false);
  const socket = useRef(null);
  const scroller = useRef(null);

  // The websocket echoes our own messages back, so de-duplicate on id.
  const appendMessage = useCallback((message) => {
    setMessages((current) =>
      current.some((item) => item.id === message.id) ? current : [...current, message]
    );
  }, []);

  useEffect(() => {
    if (!bookingId) return undefined;
    let active = true;

    endpoints
      .messages(bookingId)
      .then(({ data }) => active && setMessages(data || []))
      .catch((error) => active && toast.error(errorMessage(error, "Could not load messages.")));

    const ws = new WebSocket(bookingSocketURL(bookingId));
    socket.current = ws;
    ws.onopen = () => active && setConnected(true);
    ws.onclose = () => active && setConnected(false);
    ws.onerror = () => active && setConnected(false);
    ws.onmessage = (event) => {
      try {
        appendMessage(JSON.parse(event.data));
      } catch {
        /* ignore malformed frames */
      }
    };

    return () => {
      active = false;
      ws.close();
      socket.current = null;
    };
  }, [bookingId, appendMessage, toast]);

  useEffect(() => {
    scroller.current?.scrollTo({ top: scroller.current.scrollHeight, behavior: "smooth" });
  }, [messages]);

  async function send(event) {
    event?.preventDefault();
    const text = content.trim();
    if (!text || sending) return;

    // Prefer the socket, but fall back to REST so a dropped connection does not
    // silently swallow the message.
    if (socket.current?.readyState === WebSocket.OPEN) {
      socket.current.send(JSON.stringify({ content: text }));
      setContent("");
      return;
    }
    setSending(true);
    try {
      const { data } = await endpoints.sendMessage(bookingId, text);
      appendMessage(data);
      setContent("");
    } catch (error) {
      toast.error(errorMessage(error, "Could not send message."));
    } finally {
      setSending(false);
    }
  }

  return (
    <div className="flex h-[32rem] flex-col overflow-hidden rounded-3xl border border-brand-border bg-brand-surface shadow-sm">
      <div className="flex items-center justify-between border-b border-brand-border px-6 py-4">
        <h3 className="font-display text-lg font-bold text-brand-dark">Messages</h3>
        <span className="flex items-center gap-2 text-xs font-semibold text-brand-muted">
          <span className={`h-2 w-2 rounded-full ${connected ? "bg-green-500" : "bg-brand-border"}`} />
          {connected ? "Live" : "Offline"}
        </span>
      </div>

      <div ref={scroller} className="flex-1 space-y-3 overflow-y-auto p-6">
        {messages.length === 0 && (
          <p className="py-12 text-center text-sm text-brand-muted">
            No messages yet. Say hello to get started.
          </p>
        )}
        {messages.map((message) => {
          const mine = message.sender_id === user?.id;
          return (
            <div key={message.id} className={`flex ${mine ? "justify-end" : "justify-start"}`}>
              <div
                className={`max-w-[75%] rounded-2xl px-4 py-3 ${
                  mine
                    ? "rounded-br-sm bg-brand-primary text-white"
                    : "rounded-bl-sm bg-brand-cream text-brand-dark"
                }`}
              >
                {!mine && (
                  <p className="text-xs font-semibold text-brand-muted">{message.sender?.full_name}</p>
                )}
                <p className="whitespace-pre-wrap break-words">{message.content}</p>
                <p className={`mt-1 text-xs ${mine ? "text-white/70" : "text-brand-muted"}`}>
                  {new Date(message.created_at).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}
                </p>
              </div>
            </div>
          );
        })}
      </div>

      <form onSubmit={send} className="flex items-center gap-3 border-t border-brand-border p-4">
        <input
          className="input flex-1 py-3"
          value={content}
          onChange={(event) => setContent(event.target.value)}
          placeholder="Type a message..."
          aria-label="Message"
        />
        <button
          type="submit"
          disabled={!content.trim() || sending}
          aria-label="Send message"
          className="rounded-xl bg-brand-primary p-3 text-white transition-all hover:bg-brand-primary/90 active:scale-95 disabled:opacity-40"
        >
          <Send size={18} />
        </button>
      </form>
    </div>
  );
}
