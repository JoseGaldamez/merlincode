import React from 'react';
import { CategoryItem, SettingsCategoryType } from '../types';

interface SettingsNavProps {
  categories: CategoryItem[];
  activeCategory: SettingsCategoryType;
  onSelectCategory: (id: SettingsCategoryType) => void;
  projectsCount: number;
}

export const SettingsNav: React.FC<SettingsNavProps> = ({
  categories,
  activeCategory,
  onSelectCategory,
  projectsCount,
}) => {
  return (
    <nav
      className="w-64 min-w-[256px] bg-[#0b0f11] border-r border-border-subtle/50 flex flex-col p-4 shrink-0 select-none"
      aria-label="Categorías de configuración"
    >
      <div className="px-3 py-1.5 mb-2">
        <span className="text-xs font-mono font-semibold tracking-wider text-content-dim uppercase">
          Preferencias
        </span>
      </div>

      <ul className="flex flex-col gap-1.5 list-none p-0 m-0">
        {categories.map((cat) => {
          const isActive = cat.id === activeCategory;
          return (
            <li key={cat.id}>
              <button
                type="button"
                className={`w-full flex items-center gap-3 px-3.5 py-2.5 rounded-lg text-sm transition-all cursor-pointer text-left ${
                  isActive
                    ? 'bg-[#152329] text-accent-primary font-semibold shadow-sm'
                    : 'text-content-muted hover:bg-[#12181b] hover:text-content-body'
                }`}
                onClick={() => onSelectCategory(cat.id)}
              >
                <span className={`shrink-0 ${isActive ? 'text-accent-primary' : 'text-content-dim'}`}>
                  {cat.icon}
                </span>
                <span className="flex-1 truncate">{cat.label}</span>
                {cat.id === 'projects' && (
                  <span className="text-xs font-mono px-2 py-0.5 rounded bg-[#1b2b32] text-accent-primary font-medium">
                    {projectsCount}
                  </span>
                )}
              </button>
            </li>
          );
        })}
      </ul>
    </nav>
  );
};
