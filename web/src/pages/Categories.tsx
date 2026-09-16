import React, { useState, useEffect } from 'react';
import { Plus, FolderTree, Edit, Trash2, Layers, AlertCircle } from 'lucide-react';
import { api, Category } from '../api';

export const Categories: React.FC = () => {
  const [categories, setCategories] = useState<Category[]>([]);
  const [loading, setLoading] = useState(true);
  const [name, setName] = useState('');
  const [sortOrder, setSortOrder] = useState(0);
  const [editingCategory, setEditingCategory] = useState<Category | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadCategories();
  }, []);

  const loadCategories = async () => {
    setLoading(true);
    try {
      const data = await api.getCategories();
      setCategories(data || []);
    } catch (err: any) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;

    setError(null);
    try {
      if (editingCategory) {
        await api.updateCategory(editingCategory.id, {
          name: name.trim(),
          sort_order: Number(sortOrder) || 0,
        });
      } else {
        await api.createCategory({
          name: name.trim(),
          sort_order: Number(sortOrder) || 0,
        });
      }
      setName('');
      setSortOrder(0);
      setEditingCategory(null);
      loadCategories();
    } catch (err: any) {
      setError(err.message || 'Failed to save category');
    }
  };

  const handleEdit = (cat: Category) => {
    setEditingCategory(cat);
    setName(cat.name);
    setSortOrder(cat.sort_order);
  };

  const handleDelete = async (cat: Category) => {
    if (!confirm(`Delete category "${cat.name}"?`)) return;
    try {
      await api.deleteCategory(cat.id);
      loadCategories();
    } catch (err: any) {
      alert(err.message || 'Failed to delete category');
    }
  };

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-white tracking-tight flex items-center space-x-2">
            <FolderTree className="w-6 h-6 text-indigo-400" />
            <span>IPTV Categories</span>
          </h1>
          <p className="text-xs text-slate-400 mt-1">
            Organize channels into IPTV group-titles supported across players.
          </p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* Left Form */}
        <div className="bg-slate-900 border border-slate-800 rounded-3xl p-6 shadow-xl h-fit">
          <h3 className="text-base font-bold text-white mb-4 flex items-center space-x-2">
            <Layers className="w-4 h-4 text-indigo-400" />
            <span>{editingCategory ? 'Edit Category' : 'Create New Category'}</span>
          </h3>

          {error && (
            <div className="mb-4 p-3 bg-red-500/10 border border-red-500/20 rounded-xl flex items-center space-x-2 text-red-400 text-xs">
              <AlertCircle className="w-4 h-4 shrink-0" />
              <span>{error}</span>
            </div>
          )}

          <form onSubmit={handleSave} className="space-y-4">
            <div>
              <label className="block text-xs font-semibold text-slate-300 mb-1.5">
                Category Name <span className="text-red-400">*</span>
              </label>
              <input
                type="text"
                required
                placeholder="e.g. Sports, News, Movies"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2 text-sm text-slate-200 placeholder-slate-600 focus:outline-none focus:border-indigo-500"
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-300 mb-1.5">
                Sort Order (Lower numbers appear first)
              </label>
              <input
                type="number"
                value={sortOrder}
                onChange={(e) => setSortOrder(Number(e.target.value))}
                className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2 text-sm text-slate-200 focus:outline-none focus:border-indigo-500"
              />
            </div>

            <div className="flex items-center space-x-3 pt-2">
              {editingCategory && (
                <button
                  type="button"
                  onClick={() => {
                    setEditingCategory(null);
                    setName('');
                    setSortOrder(0);
                  }}
                  className="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-xl text-xs font-semibold"
                >
                  Cancel
                </button>
              )}
              <button
                type="submit"
                className="flex-1 bg-indigo-600 hover:bg-indigo-500 text-white font-semibold py-2 px-4 rounded-xl text-xs shadow-lg shadow-indigo-600/20 transition-all flex items-center justify-center space-x-2"
              >
                <Plus className="w-4 h-4" />
                <span>{editingCategory ? 'Save Changes' : 'Create Category'}</span>
              </button>
            </div>
          </form>
        </div>

        {/* Right Categories List */}
        <div className="lg:col-span-2">
          <div className="bg-slate-900 border border-slate-800 rounded-3xl overflow-hidden shadow-xl">
            <div className="p-4 border-b border-slate-800 bg-slate-900/80">
              <span className="text-xs font-semibold text-slate-400">Total Categories: {categories.length}</span>
            </div>

            {loading ? (
              <div className="p-12 text-center text-slate-500 text-sm">Loading categories...</div>
            ) : categories.length === 0 ? (
              <div className="p-12 text-center text-slate-500 text-sm italic">
                No categories created yet. Create one using the form on the left.
              </div>
            ) : (
              <div className="divide-y divide-slate-800/60">
                {categories.map((cat) => (
                  <div key={cat.id} className="p-4 flex items-center justify-between hover:bg-slate-800/30 transition-colors">
                    <div className="flex items-center space-x-3">
                      <div className="w-8 h-8 rounded-lg bg-indigo-500/10 text-indigo-400 flex items-center justify-center font-bold text-xs">
                        {cat.sort_order}
                      </div>
                      <div>
                        <h4 className="text-sm font-semibold text-slate-200">{cat.name}</h4>
                        <span className="text-xs text-slate-500 font-mono">slug: {cat.slug}</span>
                      </div>
                    </div>

                    <div className="flex items-center space-x-3">
                      <span className="text-xs px-2.5 py-1 rounded-full bg-slate-800 text-slate-300 font-medium">
                        {cat.stream_count || 0} channels
                      </span>

                      <button
                        onClick={() => handleEdit(cat)}
                        className="p-1.5 text-slate-400 hover:text-indigo-400 hover:bg-indigo-500/10 rounded-lg transition-colors"
                      >
                        <Edit className="w-4 h-4" />
                      </button>

                      <button
                        onClick={() => handleDelete(cat)}
                        className="p-1.5 text-slate-400 hover:text-red-400 hover:bg-red-500/10 rounded-lg transition-colors"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};
