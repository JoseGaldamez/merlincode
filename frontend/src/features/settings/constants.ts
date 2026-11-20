export interface AIModelOption {
  id: string;
  name: string;
  badge: string;
  description: string;
  isRecommended?: boolean;
}

export interface AIProviderConfig {
  id: 'anthropic' | 'openai' | 'google' | 'deepseek';
  name: string;
  badge: string;
  tagline: string;
  officialEndpoint: string;
  keyPlaceholder: string;
  keyHelpUrl: string;
  defaultModel: string;
  brandColor?: string;
  brandBg?: string;
  brandBorder?: string;
  models: AIModelOption[];
}

export const AI_PROVIDERS: AIProviderConfig[] = [
  {
    id: 'anthropic',
    name: 'Anthropic',
    badge: 'Claude 5.5 & Fable',
    tagline: 'Razonamiento híbrido de vanguardia y análisis exhaustivo de código.',
    officialEndpoint: 'https://api.anthropic.com/v1',
    keyPlaceholder: 'sk-ant-api03-...',
    keyHelpUrl: 'https://console.anthropic.com/settings/keys',
    defaultModel: 'claude-sonnet-5-5',
    brandColor: 'text-[#E07A5F]',
    brandBg: 'bg-[#E07A5F]/10',
    brandBorder: 'border-[#E07A5F]/30',
    models: [
      {
        id: 'claude-sonnet-5-5',
        name: 'Claude Sonnet 5.5',
        badge: 'Orquestador',
        description: 'Modelo equilibrado para orquestación y desarrollo de software.',
        isRecommended: true,
      },
      {
        id: 'claude-fable-5-2',
        name: 'Claude Fable 5.2',
        badge: 'Frontier',
        description: 'Máxima capacidad cognitiva para resolución de problemas de vanguardia.',
      },
      {
        id: 'claude-opus-5-5',
        name: 'Claude Opus 5.5',
        badge: 'Razonamiento Complejo',
        description: 'Análisis profundo de arquitectura y comprensión de dependencias complejas.',
      },
      {
        id: 'claude-haiku-5-5',
        name: 'Claude Haiku 5.5',
        badge: 'Ultrarrápido',
        description: 'Velocidad de respuesta instantánea y costo ultra reducido para tareas ágiles.',
      },
    ],
  },
  {
    id: 'openai',
    name: 'OpenAI',
    badge: 'GPT-6.2 Sol & GPT-6 Luna',
    tagline: 'Modelos multimodales emblemáticos y razonamiento lógico profundo.',
    officialEndpoint: 'https://api.openai.com/v1',
    keyPlaceholder: 'sk-proj-...',
    keyHelpUrl: 'https://platform.openai.com/api-keys',
    defaultModel: 'gpt-6.2-sol',
    brandColor: 'text-[#10A37F]',
    brandBg: 'bg-[#10A37F]/10',
    brandBorder: 'border-[#10A37F]/30',
    models: [
      {
        id: 'gpt-6.2-sol',
        name: 'GPT-6.2 Sol',
        badge: 'Orquestador',
        description: 'Modelo equilibrado para orquestación y generación precisa de código.',
        isRecommended: true,
      },
      {
        id: 'gpt-6-luna',
        name: 'GPT-6 Luna',
        badge: 'Rápido & Ligero',
        description: 'Variante rápida y económica optimizada para asistencia continua.',
      },
    ],
  },
  {
    id: 'google',
    name: 'Google',
    badge: 'Gemini 4 & Flash-Lite',
    tagline: 'Ventana de contexto de gran capacidad y velocidad multimodal de Google DeepMind.',
    officialEndpoint: 'https://generativelanguage.googleapis.com/v1beta',
    keyPlaceholder: 'AIzaSy...',
    keyHelpUrl: 'https://aistudio.google.com/app/apikey',
    defaultModel: 'gemini-4-hydrogen',
    brandColor: 'text-[#4285F4]',
    brandBg: 'bg-[#4285F4]/10',
    brandBorder: 'border-[#4285F4]/30',
    models: [
      {
        id: 'gemini-4-hydrogen',
        name: 'Gemini 4 Hydrogen',
        badge: 'Orquestador',
        description: 'Modelo equilibrado para programación y uso diario de herramientas.',
        isRecommended: true,
      },
      {
        id: 'gemini-3.5-flash-lite',
        name: 'Gemini 3.5 Flash-Lite',
        badge: 'Ultrarrápido',
        description: 'Velocidad de respuesta ultrarrápida y mínima latencia para tareas ágiles.',
      },
      {
        id: 'gemini-4-neon',
        name: 'Gemini 4 Neon',
        badge: 'Razonamiento Complejo',
        description: 'Modelo para tareas complejas de ingeniería de software con respuesta rápida.',
      },
      {
        id: 'gemini-4-carbon',
        name: 'Gemini 4 Carbon',
        badge: 'Frontier',
        description: 'Modelo de mayor capacidad para razonamiento y programación compleja.',
      },
    ],
  },
  {
    id: 'deepseek',
    name: 'DeepSeek',
    badge: 'V4.1 Flash & V4 Pro',
    tagline: 'Arquitectura MoE de código abierto y razonamiento matemático avanzado.',
    officialEndpoint: 'https://api.deepseek.com',
    keyPlaceholder: 'sk-...',
    keyHelpUrl: 'https://platform.deepseek.com/api_keys',
    defaultModel: 'deepseek-flash',
    brandColor: 'text-[#4D6BFE]',
    brandBg: 'bg-[#4D6BFE]/10',
    brandBorder: 'border-[#4D6BFE]/30',
    models: [
      {
        id: 'deepseek-flash',
        name: 'DeepSeek V4.1 Flash',
        badge: 'Orquestador',
        description: 'Modelo rápido y económico para programación y uso de herramientas.',
        isRecommended: true,
      },
      {
        id: 'deepseek-v4-pro',
        name: 'DeepSeek V4 Pro',
        badge: 'Razonamiento Complejo',
        description: 'Modelo de mayor capacidad de la familia V4 para tareas complejas.',
      },
    ],
  },
];

export const DEFAULT_PROVIDER_ID = 'anthropic';

export function getProviderConfig(providerId: string): AIProviderConfig {
  const found = AI_PROVIDERS.find((p) => p.id === providerId);
  return found || AI_PROVIDERS[0];
}

export function getDefaultModelForProvider(providerId: string): string {
  const provider = getProviderConfig(providerId);
  return provider.defaultModel;
}
