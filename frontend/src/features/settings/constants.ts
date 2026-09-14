export interface AIModelOption {
  id: string;
  name: string;
  badge: string;
  description: string;
  isRecommended?: boolean;
}

export interface AIProviderPricing {
  promptPerMillion: number;
  completionPerMillion: number;
  unitLabel: string;
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
  pricing: AIProviderPricing;
  brandColor?: string;
  brandBg?: string;
  brandBorder?: string;
  models: AIModelOption[];
}

export const AI_PROVIDERS: AIProviderConfig[] = [
  {
    id: 'anthropic',
    name: 'Anthropic',
    badge: 'Claude 5 & Sonnet',
    tagline: 'Razonamiento híbrido de vanguardia y análisis exhaustivo de código.',
    officialEndpoint: 'https://api.anthropic.com/v1',
    keyPlaceholder: 'sk-ant-api03-...',
    keyHelpUrl: 'https://console.anthropic.com/settings/keys',
    defaultModel: 'claude-sonnet-5',
    pricing: {
      promptPerMillion: 3.0,
      completionPerMillion: 15.0,
      unitLabel: '$3.00 / $15.00 por 1M tokens',
    },
    brandColor: 'text-[#E07A5F]',
    brandBg: 'bg-[#E07A5F]/10',
    brandBorder: 'border-[#E07A5F]/30',
    models: [
      {
        id: 'claude-sonnet-5',
        name: 'Claude Sonnet 5',
        badge: 'Orquestador',
        description: 'Modelo equilibrado para orquestación y desarrollo de software.',
        isRecommended: true,
      },
      {
        id: 'claude-fable-5-1',
        name: 'Claude Fable 5.1',
        badge: 'Frontier',
        description: 'Máxima capacidad cognitiva para resolución de problemas de vanguardia.',
      },
      {
        id: 'claude-opus-5',
        name: 'Claude Opus 5',
        badge: 'Razonamiento Complejo',
        description: 'Análisis profundo de arquitectura y comprensión de dependencias complejas.',
      },
      {
        id: 'claude-haiku-4-5-20251001',
        name: 'Claude Haiku 4.5',
        badge: 'Ultrarrápido',
        description: 'Velocidad de respuesta instantánea y costo ultra reducido para tareas ágiles.',
      },
    ],
  },
  {
    id: 'openai',
    name: 'OpenAI',
    badge: 'GPT-5.6 & GPT-6',
    tagline: 'Modelos multimodales emblemáticos y razonamiento lógico profundo.',
    officialEndpoint: 'https://api.openai.com/v1',
    keyPlaceholder: 'sk-proj-...',
    keyHelpUrl: 'https://platform.openai.com/api-keys',
    defaultModel: 'gpt-5.6-terra',
    pricing: {
      promptPerMillion: 2.5,
      completionPerMillion: 10.0,
      unitLabel: '$2.50 / $10.00 por 1M tokens',
    },
    brandColor: 'text-[#10A37F]',
    brandBg: 'bg-[#10A37F]/10',
    brandBorder: 'border-[#10A37F]/30',
    models: [
      {
        id: 'gpt-5.6-terra',
        name: 'GPT-5.6 Terra',
        badge: 'Orquestador',
        description: 'Modelo equilibrado para orquestación y generación precisa de código.',
        isRecommended: true,
      },
      {
        id: 'gpt-6-astra',
        name: 'GPT-6 Astra',
        badge: 'Frontier',
        description: 'Capacidad de razonamiento frontier de última generación.',
      },
      {
        id: 'gpt-5.6-sol',
        name: 'GPT-5.6 Sol',
        badge: 'Razonamiento Complejo',
        description: 'Especializado en algoritmos complejos y arquitectura profunda.',
      },
      {
        id: 'gpt-5.6-luna',
        name: 'GPT-5.6 Luna',
        badge: 'Rápido & Ligero',
        description: 'Variante rápida y económica optimizada para asistencia continua.',
      },
    ],
  },
  {
    id: 'google',
    name: 'Google',
    badge: 'Gemini 3.8 & Flash',
    tagline: 'Ventana de contexto de gran capacidad y velocidad multimodal de Google DeepMind.',
    officialEndpoint: 'https://generativelanguage.googleapis.com/v1beta',
    keyPlaceholder: 'AIzaSy...',
    keyHelpUrl: 'https://aistudio.google.com/app/apikey',
    defaultModel: 'gemini-3.8-flash',
    pricing: {
      promptPerMillion: 1.25,
      completionPerMillion: 5.0,
      unitLabel: '$1.25 / $5.00 por 1M tokens',
    },
    brandColor: 'text-[#4285F4]',
    brandBg: 'bg-[#4285F4]/10',
    brandBorder: 'border-[#4285F4]/30',
    models: [
      {
        id: 'gemini-3.8-flash',
        name: 'Gemini 3.8 Flash',
        badge: 'Orquestador SOTA',
        description: 'Modelo insignia equilibrado para orquestación, asistencia continua y código.',
        isRecommended: true,
      },
      {
        id: 'gemini-3.5-flash-lite',
        name: 'Gemini 3.5 Flash-Lite',
        badge: 'Ultrarrápido',
        description: 'Velocidad de respuesta ultrarrápida y mínima latencia para tareas ágiles.',
      },
      {
        id: 'gemini-3.1-pro-preview',
        name: 'Gemini 3.1 Pro',
        badge: 'Preview Razonamiento',
        description: 'Razonamiento lógico avanzado y comprensión profunda de arquitecturas.',
      },
      {
        id: 'gemini-2.5-pro',
        name: 'Gemini 2.5 Pro',
        badge: 'Razonamiento Complejo',
        description: 'Capacidad de contexto masivo y resolución de problemas complejos.',
      },
    ],
  },
  {
    id: 'deepseek',
    name: 'DeepSeek',
    badge: 'V4 Pro & Flash',
    tagline: 'Arquitectura MoE de código abierto y razonamiento matemático avanzado.',
    officialEndpoint: 'https://api.deepseek.com',
    keyPlaceholder: 'sk-...',
    keyHelpUrl: 'https://platform.deepseek.com/api_keys',
    defaultModel: 'deepseek-v4-pro',
    pricing: {
      promptPerMillion: 0.14,
      completionPerMillion: 0.28,
      unitLabel: '$0.14 / $0.28 por 1M tokens',
    },
    brandColor: 'text-[#4D6BFE]',
    brandBg: 'bg-[#4D6BFE]/10',
    brandBorder: 'border-[#4D6BFE]/30',
    models: [
      {
        id: 'deepseek-v4-pro',
        name: 'DeepSeek V4 Pro',
        badge: 'Orquestador',
        description: 'Modelo insignia de alta capacidad para orquestación y desarrollo de código.',
        isRecommended: true,
      },
      {
        id: 'deepseek-v4-flash',
        name: 'DeepSeek V4 Flash',
        badge: 'Ultrarrápido',
        description: 'Velocidad de respuesta instantánea y costo mínimo para tareas ágiles.',
      },
      {
        id: 'deepseek-v4-flash-vision-exp',
        name: 'DeepSeek V4 Flash Vision',
        badge: 'Visión Exp',
        description: 'Variante multimodal experimental optimizada para análisis visual.',
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
