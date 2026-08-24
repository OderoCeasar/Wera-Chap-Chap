import axios from "axios";

const ACCESS_KEY = "wera_access_token";
const REFRESH_KEY = "wera_refresh_token";
const USER_KEY = "wera_user";

export const tokenStore = {
  access: () => localStorage.getItem(ACCESS_KEY),
  refresh: () => localStorage.getItem(REFRESH_KEY),
  user: () => {
    try {
      return JSON.parse(localStorage.getItem(USER_KEY) || "null");
    } catch {
      return null;
    }
  },
  save: ({ access_token, refresh_token, user }) => {
    if (access_token) localStorage.setItem(ACCESS_KEY, access_token);
    if (refresh_token) localStorage.setItem(REFRESH_KEY, refresh_token);
    if (user) localStorage.setItem(USER_KEY, JSON.stringify(user));
  },
  clear: () => {
    [ACCESS_KEY, REFRESH_KEY, USER_KEY].forEach((key) => localStorage.removeItem(key));
  }
};

const baseURL = import.meta.env.VITE_API_URL || "/api";

export const api = axios.create({ baseURL });

api.interceptors.request.use((config) => {
  const token = tokenStore.access();
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});

// Broadcast a forced logout so AuthContext can react without a hard reload.
function forceLogout() {
  tokenStore.clear();
  window.dispatchEvent(new Event("wera:unauthorized"));
}

// A single in-flight refresh shared by every request that got a 401, so a burst
// of parallel calls does not trigger a burst of refreshes.
let refreshing = null;

async function refreshAccessToken() {
  const refresh_token = tokenStore.refresh();
  if (!refresh_token) throw new Error("no refresh token");
  const { data } = await axios.post(`${baseURL}/auth/refresh`, { refresh_token });
  tokenStore.save({ access_token: data.access_token });
  return data.access_token;
}

api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const original = error.config;
    const isAuthCall = original?.url?.includes("/auth/");

    if (error.response?.status !== 401 || original?._retried || isAuthCall) {
      return Promise.reject(error);
    }

    original._retried = true;
    try {
      refreshing = refreshing || refreshAccessToken().finally(() => { refreshing = null; });
      const token = await refreshing;
      original.headers.Authorization = `Bearer ${token}`;
      return api(original);
    } catch (refreshError) {
      forceLogout();
      return Promise.reject(refreshError);
    }
  }
);

/** Pull a readable message out of an axios error. */
export function errorMessage(error, fallback = "Something went wrong. Please try again.") {
  return error?.response?.data?.error || error?.message || fallback;
}

/** Build the websocket URL for a booking room, honouring VITE_API_URL. */
export function bookingSocketURL(bookingId) {
  const token = tokenStore.access();
  const origin = /^https?:\/\//.test(baseURL)
    ? baseURL.replace(/\/api\/?$/, "")
    : window.location.origin;
  return `${origin.replace(/^http/, "ws")}/ws/booking/${bookingId}?token=${token}`;
}

export const endpoints = {
  // auth
  login: (payload) => api.post("/auth/login", payload),
  register: (payload) => api.post("/auth/register", payload),
  logout: () => api.post("/auth/logout"),

  // users
  me: () => api.get("/users/me"),
  updateMe: (payload) => api.put("/users/me", payload),
  changePassword: (payload) => api.put("/users/me/password", payload),

  // catalogue
  categories: () => api.get("/categories"),
  taskers: (params) => api.get("/taskers", { params }),
  tasker: (id) => api.get(`/taskers/${id}`),

  // tasker self-service
  myTaskerProfile: () => api.get("/taskers/me"),
  updateTaskerProfile: (payload) => api.put("/taskers/profile", payload),
  setAvailability: (slots) => api.post("/taskers/availability", slots),
  myTaskerBookings: () => api.get("/taskers/me/bookings"),
  myApplications: () => api.get("/taskers/me/applications"),

  // tasks
  tasks: (params) => api.get("/tasks", { params }),
  myTasks: () => api.get("/tasks/my"),
  task: (id) => api.get(`/tasks/${id}`),
  createTask: (payload) => api.post("/tasks", payload),
  updateTask: (id, payload) => api.put(`/tasks/${id}`, payload),
  cancelTask: (id) => api.delete(`/tasks/${id}`),
  matches: (id) => api.post(`/tasks/${id}/matches`),
  applyToTask: (id, payload) => api.post(`/tasks/${id}/apply`, payload),
  acceptApplication: (taskId, appId) => api.put(`/tasks/${taskId}/applications/${appId}/accept`),
  rejectApplication: (taskId, appId) => api.put(`/tasks/${taskId}/applications/${appId}/reject`),

  // bookings
  bookings: () => api.get("/bookings"),
  booking: (id) => api.get(`/bookings/${id}`),
  startBooking: (id) => api.put(`/bookings/${id}/start`),
  completeBooking: (id) => api.put(`/bookings/${id}/complete`),
  cancelBooking: (id) => api.put(`/bookings/${id}/cancel`),

  // messaging
  messages: (bookingId) => api.get(`/messages/booking/${bookingId}`),
  sendMessage: (bookingId, content) => api.post(`/messages/booking/${bookingId}`, { content }),

  // reviews
  taskerReviews: (taskerId) => api.get(`/reviews/tasker/${taskerId}`),
  bookingReviews: (bookingId) => api.get(`/reviews/booking/${bookingId}`),
  createReview: (bookingId, payload) => api.post(`/reviews/booking/${bookingId}`, payload),

  // payments
  payment: (bookingId) => api.get(`/payments/booking/${bookingId}`),
  initiatePayment: (bookingId, payload) => api.post(`/payments/booking/${bookingId}/initiate`, payload),
  confirmPayment: (bookingId) => api.post(`/payments/booking/${bookingId}/confirm`),
  tipPayment: (bookingId, tip_amount) => api.post(`/payments/booking/${bookingId}/tip`, { tip_amount })
};
