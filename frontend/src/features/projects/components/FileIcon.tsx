import React from 'react';
import { IconFile, IconFileCode } from '../../../components/Icons';

interface FileIconProps {
  name: string;
}

export const FileIcon: React.FC<FileIconProps> = ({ name }) => {
  const ext = name.split('.').pop()?.toLowerCase() || '';

  if (['ts', 'tsx', 'js', 'jsx', 'go', 'py', 'rs', 'c', 'cpp', 'java', 'php', 'rb'].includes(ext)) {
    return <IconFileCode size={13} className="text-accent-primary shrink-0" />;
  }
  if (['json', 'yaml', 'yml', 'toml', 'xml', 'env'].includes(ext)) {
    return <IconFile size={13} className="text-amber-400 shrink-0" />;
  }
  if (['md', 'txt', 'rst', 'markdown'].includes(ext)) {
    return <IconFile size={13} className="text-sky-400 shrink-0" />;
  }
  if (['png', 'jpg', 'jpeg', 'gif', 'svg', 'webp', 'ico'].includes(ext)) {
    return <IconFile size={13} className="text-purple-400 shrink-0" />;
  }
  return <IconFile size={13} className="text-content-dim shrink-0" />;
};
