export namespace main {
	
	export class CheatState {
	    name: string;
	    description: string;
	    enabled: boolean;
	    input: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CheatState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.enabled = source["enabled"];
	        this.input = source["input"];
	    }
	}
	export class ProcessInfo {
	    pid: number;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new ProcessInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pid = source["pid"];
	        this.name = source["name"];
	    }
	}
	export class ScanResult {
	    addresses: string[];
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new ScanResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.addresses = source["addresses"];
	        this.count = source["count"];
	    }
	}
	export class TrainerStatus {
	    game: string;
	    exe: string;
	    version: string;
	    pid: number;
	    connected: boolean;
	    cheats: CheatState[];
	
	    static createFrom(source: any = {}) {
	        return new TrainerStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.game = source["game"];
	        this.exe = source["exe"];
	        this.version = source["version"];
	        this.pid = source["pid"];
	        this.connected = source["connected"];
	        this.cheats = this.convertValues(source["cheats"], CheatState);
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
	export class TrainerSummary {
	    filename: string;
	    game: string;
	    exe: string;
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new TrainerSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filename = source["filename"];
	        this.game = source["game"];
	        this.exe = source["exe"];
	        this.version = source["version"];
	    }
	}

}

