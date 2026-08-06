import { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuthStore } from '@/stores/authStore';

function Dashboard() {
  const navigate = useNavigate();
  const user = useAuthStore((state) => state.user);

  useEffect(() => {
    if (user?.role === 'admin') {
      navigate('/admin/users', { replace: true });
    } else {
      navigate('/accounts', { replace: true });
    }
  }, [user, navigate]);

  return null;
}

export default Dashboard;
