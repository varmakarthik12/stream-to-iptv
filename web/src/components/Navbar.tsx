import React, { useState, useEffect } from 'react';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import { Tv, FolderTree, Image as ImageIcon, Calendar, Settings as SettingsIcon, LogOut, Radio, Activity, Menu, X } from 'lucide-react';
import { api, User } from '../api';

interface NavbarProps {
  user: User | null;
  onLogout: () => void;
}

export const Navbar: React.FC<NavbarProps> = ({ user, onLogout }) => {
  const location = useLocation();
  const navigate = useNavigate();
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  useEffect(() => {
    setMobileMenuOpen(false);
  }, [location.pathname]);

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

  const isItemActive = (path: string) => {
    if (path === '/') {
      return (
        location.pathname === '/' ||
        location.pathname === '/streams' ||
        location.pathname.startsWith('/streams/') ||
        location.pathname === '/guide' ||
        location.pathname === '/setup-guide'
      );
    }
    return location.pathname === path || location.pathname.startsWith(path + '/');
  };

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
                const isActive = isItemActive(item.path);
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

          <div className="flex items-center space-x-2 sm:space-x-4">
            {user && (
              <div className="flex items-center space-x-3">
                <span className="text-xs font-medium text-slate-400 hidden sm:inline-block">
                  Signed in as <strong className="text-slate-200">{user.username}</strong>
                </span>
                <button
                  onClick={handleLogout}
                  title="Sign out"
                  className="hidden md:flex items-center space-x-1.5 px-3 py-1.5 rounded-lg text-xs font-medium text-slate-400 hover:text-red-400 hover:bg-red-500/10 border border-transparent hover:border-red-500/20 transition-colors"
                >
                  <LogOut className="w-3.5 h-3.5" />
                  <span className="hidden sm:inline">Logout</span>
                </button>
              </div>
            )}

            {/* Mobile Hamburger Menu Toggle Button */}
            <button
              type="button"
              onClick={() => setMobileMenuOpen((prev) => !prev)}
              className="md:hidden p-2 rounded-xl text-slate-400 hover:text-white hover:bg-slate-800/80 border border-slate-800 transition-colors focus:outline-none focus:ring-2 focus:ring-indigo-500/50"
              aria-label={mobileMenuOpen ? 'Close navigation menu' : 'Open navigation menu'}
              aria-expanded={mobileMenuOpen}
            >
              {mobileMenuOpen ? <X className="w-5 h-5 text-slate-200" /> : <Menu className="w-5 h-5 text-slate-200" />}
            </button>
          </div>
        </div>
      </div>

      {/* Mobile Navigation Drawer / Dropdown */}
      {mobileMenuOpen && (
        <div
          role="navigation"
          aria-label="Mobile navigation"
          className="md:hidden border-b border-slate-800 bg-slate-900/95 backdrop-blur-md px-4 pt-2 pb-4 space-y-1 shadow-2xl animate-in slide-in-from-top-2 duration-200"
        >
          {navItems.map((item) => {
            const Icon = item.icon;
            const isActive = isItemActive(item.path);
            return (
              <Link
                key={item.path}
                to={item.path}
                onClick={() => setMobileMenuOpen(false)}
                className={`flex items-center space-x-3 px-3.5 py-2.5 rounded-xl text-sm font-medium transition-colors ${
                  isActive
                    ? 'bg-indigo-600/20 text-indigo-400 border border-indigo-500/30'
                    : 'text-slate-300 hover:text-white hover:bg-slate-800/60'
                }`}
              >
                <Icon className="w-4 h-4 shrink-0" />
                <span>{item.label}</span>
              </Link>
            );
          })}

          {user && (
            <div className="pt-3 mt-3 border-t border-slate-800/80 flex items-center justify-between px-1">
              <div className="flex flex-col">
                <span className="text-[10px] text-slate-500 uppercase tracking-wider font-semibold">Logged in as</span>
                <span className="text-xs font-semibold text-slate-200">{user.username}</span>
              </div>
              <button
                onClick={() => {
                  setMobileMenuOpen(false);
                  handleLogout();
                }}
                className="flex items-center space-x-1.5 px-3 py-1.5 rounded-lg text-xs font-medium text-red-400 bg-red-500/10 hover:bg-red-500/20 border border-red-500/20 transition-colors"
              >
                <LogOut className="w-3.5 h-3.5" />
                <span>Logout</span>
              </button>
            </div>
          )}
        </div>
      )}
    </nav>
  );
};
