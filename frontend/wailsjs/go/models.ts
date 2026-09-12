export namespace main {
	
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

