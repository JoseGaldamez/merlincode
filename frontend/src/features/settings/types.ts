import React from 'react';
import { Project, AgentTelemetry } from '../../types';

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
  apiKey: string;
  apiEndpoint: string;
  streaming: boolean;
  theme: string;
  language: string;
  autoSaveSessions: boolean;
  telemetryEnabled: boolean;
}

export interface SettingsModalProps {
  isOpen: boolean;
  onClose: () => void;
  projects: Project[];
  activeProjectId: string;
  onSelectProject: (id: string) => void;
  onAddProject?: (name: string) => void;
  onDeleteProject?: (id: string) => void;
  onOpenFolder?: () => void;
  telemetry: AgentTelemetry;
  onUpdateTelemetry: (updated: Partial<AgentTelemetry>) => void;
}
