import React from 'react';
import { Project } from '../../types';

export type SettingsCategoryType = 'general' | 'ai' | 'projects' | 'shortcuts' | 'privacy';

export interface CategoryItem {
  id: SettingsCategoryType;
  label: string;
  description: string;
  icon: React.ReactNode;
}

export interface SettingsState {
  modelProvider: string;
  model: string;
  temperature: number;
  streaming: boolean;
  theme: string;
  language: string;
  autoSaveSessions: boolean;
  telemetryEnabled: boolean;
}

export interface SettingsModalProps {
  onClose: () => void;
  projects: Project[];
  activeProjectId: string;
  onSelectProject: (id: string) => void;
  onAddProject?: (name: string) => void;
  onDeleteProject?: (id: string) => void;
  onOpenFolder?: () => void;
  initialSettings: SettingsState;
  onSave: (settings: SettingsState) => void;
}
