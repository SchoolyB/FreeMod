export namespace main {
	
	export class AllTypesResult {
	    Int32: number;
	    Int64: number;
	    Float32: number;
	    Valid: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AllTypesResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Int32 = source["Int32"];
	        this.Int64 = source["Int64"];
	        this.Float32 = source["Float32"];
	        this.Valid = source["Valid"];
	    }
	}
	export class CheatState {
	    Name: string;
	    Description: string;
	    Enabled: boolean;
	    Input: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CheatState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.Description = source["Description"];
	        this.Enabled = source["Enabled"];
	        this.Input = source["Input"];
	    }
	}
	export class ProcessInfo {
	    PID: number;
	    Name: string;
	
	    static createFrom(source: any = {}) {
	        return new ProcessInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.PID = source["PID"];
	        this.Name = source["Name"];
	    }
	}
	export class ScanResult {
	    Addresses: string[];
	    Count: number;
	
	    static createFrom(source: any = {}) {
	        return new ScanResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Addresses = source["Addresses"];
	        this.Count = source["Count"];
	    }
	}
	export class Settings {
	    AutoConnect: boolean;
	    FreezeIntervalMs: number;
	    TrainerDir: string;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.AutoConnect = source["AutoConnect"];
	        this.FreezeIntervalMs = source["FreezeIntervalMs"];
	        this.TrainerDir = source["TrainerDir"];
	    }
	}
	export class TrainerStatus {
	    Game: string;
	    Exe: string;
	    Version: string;
	    PID: number;
	    Connected: boolean;
	    Cheats: CheatState[];
	
	    static createFrom(source: any = {}) {
	        return new TrainerStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Game = source["Game"];
	        this.Exe = source["Exe"];
	        this.Version = source["Version"];
	        this.PID = source["PID"];
	        this.Connected = source["Connected"];
	        this.Cheats = this.convertValues(source["Cheats"], CheatState);
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
	    Filename: string;
	    Game: string;
	    Exe: string;
	    Version: string;
	
	    static createFrom(source: any = {}) {
	        return new TrainerSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Filename = source["Filename"];
	        this.Game = source["Game"];
	        this.Exe = source["Exe"];
	        this.Version = source["Version"];
	    }
	}

}

