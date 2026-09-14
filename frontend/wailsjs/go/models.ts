export namespace domain {
	
	export class ChatMessage {
	    role: string;
	    content: string;
	
	    static createFrom(source: any = {}) {
	        return new ChatMessage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.role = source["role"];
	        this.content = source["content"];
	    }
	}
	export class ChatMessageRecord {
	    id: string;
	    sessionId: string;
	    role: string;
	    content: string;
	    thoughtChain?: string;
	    providerId?: string;
	    modelId?: string;
	    tokensPrompt?: number;
	    tokensCompletion?: number;
	    durationSeconds?: number;
	    feedback?: string;
	    status?: string;
	    // Go type: time
	    createdAt: any;
	
	    static createFrom(source: any = {}) {
	        return new ChatMessageRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.sessionId = source["sessionId"];
	        this.role = source["role"];
	        this.content = source["content"];
	        this.thoughtChain = source["thoughtChain"];
	        this.providerId = source["providerId"];
	        this.modelId = source["modelId"];
	        this.tokensPrompt = source["tokensPrompt"];
	        this.tokensCompletion = source["tokensCompletion"];
	        this.durationSeconds = source["durationSeconds"];
	        this.feedback = source["feedback"];
	        this.status = source["status"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ChatStreamRequest {
	    sessionId: string;
	    messageId: string;
	    userMessageId?: string;
	    providerId: string;
	    modelId?: string;
	    prompt: string;
	    history?: ChatMessage[];
	
	    static createFrom(source: any = {}) {
	        return new ChatStreamRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionId = source["sessionId"];
	        this.messageId = source["messageId"];
	        this.userMessageId = source["userMessageId"];
	        this.providerId = source["providerId"];
	        this.modelId = source["modelId"];
	        this.prompt = source["prompt"];
	        this.history = this.convertValues(source["history"], ChatMessage);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FileItem {
	    name: string;
	    path: string;
	    isDir: boolean;
	    size: number;
	
	    static createFrom(source: any = {}) {
	        return new FileItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.isDir = source["isDir"];
	        this.size = source["size"];
	    }
	}
	export class FileNode {
	    name: string;
	    relPath: string;
	    fullPath: string;
	    isDir: boolean;
	    size: number;
	    children?: FileNode[];
	
	    static createFrom(source: any = {}) {
	        return new FileNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.relPath = source["relPath"];
	        this.fullPath = source["fullPath"];
	        this.isDir = source["isDir"];
	        this.size = source["size"];
	        this.children = this.convertValues(source["children"], FileNode);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PanelsState {
	    leftOpen: boolean;
	    rightOpen: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PanelsState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.leftOpen = source["leftOpen"];
	        this.rightOpen = source["rightOpen"];
	    }
	}
	export class ProjectInfo {
	    id: string;
	    name: string;
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new ProjectInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.path = source["path"];
	    }
	}
	export class ProviderStatus {
	    providerId: string;
	    configured: boolean;
	    verified: boolean;
	    verifiedAt?: string;
	    accountInfo?: string;
	    maskedKey?: string;
	    models?: string[];
	    supportsAdminKey: boolean;
	    hasAdminKey: boolean;
	    maskedAdminKey?: string;
	
	    static createFrom(source: any = {}) {
	        return new ProviderStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.providerId = source["providerId"];
	        this.configured = source["configured"];
	        this.verified = source["verified"];
	        this.verifiedAt = source["verifiedAt"];
	        this.accountInfo = source["accountInfo"];
	        this.maskedKey = source["maskedKey"];
	        this.models = source["models"];
	        this.supportsAdminKey = source["supportsAdminKey"];
	        this.hasAdminKey = source["hasAdminKey"];
	        this.maskedAdminKey = source["maskedAdminKey"];
	    }
	}
	export class ProviderUsageResult {
	    available: boolean;
	    message: string;
	    tokensPrompt?: number;
	    tokensCompletion?: number;
	    costUsd?: number;
	    balanceText?: string;
	
	    static createFrom(source: any = {}) {
	        return new ProviderUsageResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.message = source["message"];
	        this.tokensPrompt = source["tokensPrompt"];
	        this.tokensCompletion = source["tokensCompletion"];
	        this.costUsd = source["costUsd"];
	        this.balanceText = source["balanceText"];
	    }
	}
	export class ProviderValidationResult {
	    valid: boolean;
	    message: string;
	    accountInfo?: string;
	    models?: string[];
	
	    static createFrom(source: any = {}) {
	        return new ProviderValidationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.valid = source["valid"];
	        this.message = source["message"];
	        this.accountInfo = source["accountInfo"];
	        this.models = source["models"];
	    }
	}
	export class Session {
	    id: string;
	    title: string;
	    projectId?: string;
	    projectPath?: string;
	    messagesCount: number;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Session(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.projectId = source["projectId"];
	        this.projectPath = source["projectPath"];
	        this.messagesCount = source["messagesCount"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TestMessageResult {
	    success: boolean;
	    message: string;
	    responseText?: string;
	    model?: string;
	
	    static createFrom(source: any = {}) {
	        return new TestMessageResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.message = source["message"];
	        this.responseText = source["responseText"];
	        this.model = source["model"];
	    }
	}
	export class WindowState {
	    width: number;
	    height: number;
	    maximised: boolean;
	    left_panel_open: boolean;
	    right_panel_open: boolean;
	
	    static createFrom(source: any = {}) {
	        return new WindowState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.width = source["width"];
	        this.height = source["height"];
	        this.maximised = source["maximised"];
	        this.left_panel_open = source["left_panel_open"];
	        this.right_panel_open = source["right_panel_open"];
	    }
	}

}

