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
    badge: 'Claude 3.7 & 3.5',
    tagline: 'Razonamiento híbrido de vanguardia y análisis exhaustivo de código.',
    officialEndpoint: 'https://api.anthropic.com/v1',
    keyPlaceholder: 'sk-ant-api03-...',
    keyHelpUrl: 'https://console.anthropic.com/settings/keys',
    defaultModel: 'claude-3-7-sonnet-latest',
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
        id: 'claude-3-7-sonnet-latest',
        name: 'Claude 3.7 Sonnet',
        badge: 'Híbrido SOTA',
        description: 'Modelo insignia con razonamiento estándar y extendido. Máxima inteligencia en código.',
        isRecommended: true,
      },
      {
        id: 'claude-3-5-sonnet-latest',
        name: 'Claude 3.5 Sonnet',
        badge: 'Recomendado Código',
        description: 'Punto de referencia de la industria en generación y refactorización precisa de código.',
      },
      {
        id: 'claude-3-5-haiku-latest',
        name: 'Claude 3.5 Haiku',
        badge: 'Ultrarrápido',
        description: 'Velocidad de respuesta instantánea y costo ultra reducido para tareas ágiles.',
      },
      {
        id: 'claude-3-opus-20240229',
        name: 'Claude 3 Opus',
        badge: 'Razonamiento Complejo',
        description: 'Análisis profundo de arquitectura y comprensión de dependencias complejas.',
      },
    ],
  },
  {
    id: 'openai',
    name: 'OpenAI',
    badge: 'GPT-4o & o-Series',
    tagline: 'Modelos multimodales emblemáticos y razonamiento lógico profundo.',
    officialEndpoint: 'https://api.openai.com/v1',
    keyPlaceholder: 'sk-proj-...',
    keyHelpUrl: 'https://platform.openai.com/api-keys',
    defaultModel: 'gpt-4o',
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
        id: 'gpt-4o',
        name: 'GPT-4o',
        badge: 'Omni Insignia',
        description: 'Modelo omnimodal versátil con balance óptimo entre velocidad, precisión y contexto.',
        isRecommended: true,
      },
      {
        id: 'gpt-4o-mini',
        name: 'GPT-4o mini',
        badge: 'Rápido & Ligero',
        description: 'Variante rápida y económica optimizada para asistencia y consultas continuas.',
      },
      {
        id: 'o3-mini',
        name: 'o3-mini',
        badge: 'STEM & Razonamiento',
        description: 'Especializado en razonamiento matemático y resolución rigurosa de algoritmos.',
      },
      {
        id: 'o1',
        name: 'o1',
        badge: 'Pensamiento Profundo',
        description: 'Genera cadenas de pensamiento extensas para problemas complejos de arquitectura.',
      },
    ],
  },
  {
    id: 'google',
    name: 'Google',
    badge: 'Gemini 2.5 & Flash',
    tagline: 'Ventana de contexto de gran capacidad y velocidad multimodal de Google DeepMind.',
    officialEndpoint: 'https://generativelanguage.googleapis.com/v1beta',
    keyPlaceholder: 'AIzaSy...',
    keyHelpUrl: 'https://aistudio.google.com/app/apikey',
    defaultModel: 'gemini-2.5-pro',
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
        id: 'gemini-2.5-pro',
        name: 'Gemini 2.5 Pro',
        badge: 'SOTA Razonamiento',
        description: 'Modelo más avanzado de Google para comprensión de código y razonamiento lógico.',
        isRecommended: true,
      },
      {
        id: 'gemini-2.0-flash',
        name: 'Gemini 2.0 Flash',
        badge: 'Baja Latencia',
        description: 'Rendimiento multimodal de última generación a velocidad y latencia ultra reducida.',
      },
      {
        id: 'gemini-1.5-pro',
        name: 'Gemini 1.5 Pro',
        badge: '2M Contexto',
        description: 'Ventana masiva de contexto para analizar repositorios enteros de código de una sola vez.',
      },
      {
        id: 'gemini-1.5-flash',
        name: 'Gemini 1.5 Flash',
        badge: 'Alta Frecuencia',
        description: 'Solución ligera y eficiente para tareas directas y asistencia continua en vivo.',
      },
    ],
  },
  {
    id: 'deepseek',
    name: 'DeepSeek',
    badge: 'V3 & R1 MoE',
    tagline: 'Arquitectura MoE 671B de código abierto y razonamiento matemático avanzado.',
    officialEndpoint: 'https://api.deepseek.com',
    keyPlaceholder: 'sk-...',
    keyHelpUrl: 'https://platform.deepseek.com/api_keys',
    defaultModel: 'deepseek-chat',
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
        id: 'deepseek-chat',
        name: 'DeepSeek-V3',
        badge: '671B MoE General',
        description: 'Modelo de última generación con 37B activos por token. Extraordinaria habilidad en código.',
        isRecommended: true,
      },
      {
        id: 'deepseek-reasoner',
        name: 'DeepSeek-R1',
        badge: 'Razonamiento Puro',
        description: 'Entrenado mediante aprendizaje por refuerzo puro con cadena de razonamiento visible.',
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
