import React from 'react';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import { Tv, FolderTree, Image as ImageIcon, Calendar, Settings as SettingsIcon, LogOut, Radio, Activity } from 'lucide-react';
import { api, User } from '../api';

interface NavbarProps {
  user: User | null;
  onLogout: () => void;
}

export const Navbar: React.FC<NavbarProps> = ({ user, onLogout }) => {
  const location = useLocation();
  const navigate = useNavigate();

  const handleLogout = async () => {
    try {
      await api.logout();
      onLogout();
      navigate('/login');
    } catch (err) {
      console.error('Logout error', err);
    }
  };

  const navItems = [
    { label: 'Dashboard', path: '/dashboard', icon: Activity },
    { label: 'Streams', path: '/', icon: Tv },
    { label: 'Categories', path: '/categories', icon: FolderTree },
    { label: 'Logos', path: '/logos', icon: ImageIcon },
    { label: 'EPG Sources', path: '/epg', icon: Calendar },
    { label: 'Settings', path: '/settings', icon: SettingsIcon },
  ];

  return (
    <nav className="bg-slate-900/80 backdrop-blur border-b border-slate-800 sticky top-0 z-40">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex items-center justify-between h-16">
          <div className="flex items-center space-x-3">
            <Link to="/" className="flex items-center space-x-3 group">
              <div className="w-10 h-10 rounded-xl bg-gradient-to-tr from-indigo-600 to-violet-500 flex items-center justify-center shadow-lg shadow-indigo-500/20 group-hover:scale-105 transition-transform">
                <Radio className="w-5 h-5 text-white" />
              </div>
              <div>
                <span className="text-lg font-bold bg-gradient-to-r from-white to-slate-300 bg-clip-text text-transparent">
                  Stream to IPTV
                </span>
                <span className="block text-[10px] text-indigo-400 font-medium tracking-wider uppercase">
                  Management Console
                </span>
              </div>
            </Link>

            <div className="hidden md:flex items-center space-x-1 ml-8">
              {navItems.map((item) => {
                const Icon = item.icon;
                const isActive = location.pathname === item.path;
                return (
                  <Link
                    key={item.path}
                    to={item.path}
                    className={`flex items-center space-x-2 px-3.5 py-2 rounded-lg text-sm font-medium transition-colors ${
                      isActive
                        ? 'bg-indigo-600/15 text-indigo-400 border border-indigo-500/30'
                        : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/60'
                    }`}
                  >
                    <Icon className="w-4 h-4" />
                    <span>{item.label}</span>
                  </Link>
                );
              })}
            </div>
          </div>

          <div className="flex items-center space-x-4">
            {user && (
              <div className="flex items-center space-x-3">
                <span className="text-xs font-medium text-slate-400 hidden sm:inline-block">
                  Signed in as <strong className="text-slate-200">{user.username}</strong>
                </span>
                <button
                  onClick={handleLogout}
                  title="Sign out"
                  className="flex items-center space-x-1.5 px-3 py-1.5 rounded-lg text-xs font-medium text-slate-400 hover:text-red-400 hover:bg-red-500/10 border border-transparent hover:border-red-500/20 transition-colors"
                >
                  <LogOut className="w-3.5 h-3.5" />
                  <span className="hidden sm:inline">Logout</span>
                </button>
              </div>
            )}
          </div>
        </div>
      </div>
    </nav>
  );
};
