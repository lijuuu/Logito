import React, { useState } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { Button } from './ui/button';
import { Badge } from './ui/badge';
import { Menu, X, User } from 'lucide-react';

interface NavigationProps {
  currentPage: string;
  onPageChange: (page: string) => void;
}

export const Navigation: React.FC<NavigationProps> = ({ currentPage, onPageChange }) => {
  const { user, logout } = useAuth();
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);

  if (!user) return null;

  const menuItems = [
    { id: 'logs', label: 'Logs' },
    { id: 'sync-status', label: 'Sync Status' },
    { id: 'dlq', label: 'DLQ Management' },
  ];

  const toggleMobileMenu = () => {
    setIsMobileMenuOpen(!isMobileMenuOpen);
  };

  const handlePageChange = (page: string) => {
    onPageChange(page);
    setIsMobileMenuOpen(false);
  };

  return (
    <nav className="bg-white shadow-sm border-b">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex justify-between h-16">
          {/* Logo and Desktop Menu */}
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <h1 className="text-xl font-bold text-gray-900">Logito</h1>
            </div>
            {/* Desktop Navigation */}
            <div className="hidden md:ml-8 md:flex md:space-x-4">
              {menuItems.map((item) => (
                <Button
                  key={item.id}
                  variant={currentPage === item.id ? "default" : "ghost"}
                  onClick={() => onPageChange(item.id)}
                  className="text-sm"
                >
                  {item.label}
                </Button>
              ))}
            </div>
          </div>

          {/* Desktop User Info */}
          <div className="hidden md:flex md:items-center md:space-x-4">
            <div className="flex items-center space-x-2">
              <User className="h-4 w-4 text-gray-500" />
              <span className="text-sm text-gray-600 truncate max-w-32">
                {user.email}
              </span>
              <Badge variant="default">
                {user.role}
              </Badge>
            </div>
            <Button onClick={logout} variant="outline" size="sm">
              Logout
            </Button>
          </div>

          {/* Mobile menu button */}
          <div className="md:hidden flex items-center space-x-2">
            <div className="flex items-center space-x-1">
              <User className="h-4 w-4 text-gray-500" />
              <Badge variant="default" className="text-xs">
                {user.role}
              </Badge>
            </div>
            <Button
              onClick={toggleMobileMenu}
              variant="ghost"
              size="sm"
              className="p-2"
            >
              {isMobileMenuOpen ? (
                <X className="h-5 w-5" />
              ) : (
                <Menu className="h-5 w-5" />
              )}
            </Button>
          </div>
        </div>

        {/* Mobile menu */}
        {isMobileMenuOpen && (
          <div className="md:hidden">
            <div className="px-2 pt-2 pb-3 space-y-1 sm:px-3 border-t">
              {menuItems.map((item) => (
                <Button
                  key={item.id}
                  variant={currentPage === item.id ? "default" : "ghost"}
                  onClick={() => handlePageChange(item.id)}
                  className="w-full justify-start text-sm"
                >
                  {item.label}
                </Button>
              ))}
              <div className="pt-4 border-t">
                <div className="px-3 py-2">
                  <p className="text-sm text-gray-600 truncate">
                    {user.email}
                  </p>
                </div>
                <Button
                  onClick={logout}
                  variant="outline"
                  size="sm"
                  className="w-full"
                >
                  Logout
                </Button>
              </div>
            </div>
          </div>
        )}
      </div>
    </nav>
  );
};
