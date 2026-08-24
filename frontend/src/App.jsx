import { Navigate, Route, Routes, useLocation } from "react-router-dom";
import Navbar from "./components/Navbar.jsx";
import Landing from "./pages/Landing.jsx";
import Login from "./pages/Login.jsx";
import Register from "./pages/Register.jsx";
import Taskers from "./pages/Taskers.jsx";
import TaskerProfile from "./pages/TaskerProfile.jsx";
import ClientDashboard from "./pages/ClientDashboard.jsx";
import TaskerDashboard from "./pages/TaskerDashboard.jsx";
import Bookings from "./pages/Bookings.jsx";
import BookingDetail from "./pages/BookingDetail.jsx";
import NotFound from "./pages/NotFound.jsx";
import { useAuth } from "./context/AuthContext.jsx";

function Protected({ role, children }) {
  const { user, loading } = useAuth();
  const location = useLocation();

  // Wait for the stored session to be revalidated before deciding, otherwise a
  // refresh on a protected page bounces the user to /login.
  if (loading) {
    return <main className="p-16 text-center text-brand-muted">Loading your account...</main>;
  }
  if (!user) {
    return <Navigate to="/login" state={{ from: location.pathname }} replace />;
  }
  if (role && user.role !== role) {
    return <Navigate to={user.role === "tasker" ? "/dashboard/tasker" : "/dashboard/client"} replace />;
  }
  return children;
}

// Sends a signed-in user to whichever dashboard matches their role.
function DashboardRedirect() {
  const { user, loading } = useAuth();
  if (loading) return <main className="p-16 text-center text-brand-muted">Loading...</main>;
  if (!user) return <Navigate to="/login" replace />;
  return <Navigate to={user.role === "tasker" ? "/dashboard/tasker" : "/dashboard/client"} replace />;
}

export default function App() {
  return (
    <div className="flex min-h-screen flex-col">
      <Navbar />
      <div className="flex-1">
        <Routes>
          <Route path="/" element={<Landing />} />
          <Route path="/login" element={<Login />} />
          <Route path="/register" element={<Register />} />
          <Route path="/taskers" element={<Taskers />} />
          <Route path="/taskers/:id" element={<TaskerProfile />} />
          <Route path="/dashboard" element={<DashboardRedirect />} />
          <Route path="/dashboard/client" element={<Protected role="client"><ClientDashboard /></Protected>} />
          <Route path="/dashboard/tasker" element={<Protected role="tasker"><TaskerDashboard /></Protected>} />
          <Route path="/bookings" element={<Protected><Bookings /></Protected>} />
          <Route path="/bookings/:id" element={<Protected><BookingDetail /></Protected>} />
          <Route path="*" element={<NotFound />} />
        </Routes>
      </div>
    </div>
  );
}
