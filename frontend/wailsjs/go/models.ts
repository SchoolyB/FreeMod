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
	export class AllTypesScanResult {
	    Int32: ScanResult;
	    Int64: ScanResult;
	    Float32: ScanResult;
	    Float64: ScanResult;
	
	    static createFrom(source: any = {}) {
	        return new AllTypesScanResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Int32 = this.convertValues(source["Int32"], ScanResult);
	        this.Int64 = this.convertValues(source["Int64"], ScanResult);
	        this.Float32 = this.convertValues(source["Float32"], ScanResult);
	        this.Float64 = this.convertValues(source["Float64"], ScanResult);
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
	export class CheatState {
	    Name: string;
	    Description: string;
	    Enabled: boolean;
	    Input: boolean;
	    Behavior: string;
	    Value: number;
	    Trigger: number;
	
	    static createFrom(source: any = {}) {
	        return new CheatState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.Description = source["Description"];
	        this.Enabled = source["Enabled"];
	        this.Input = source["Input"];
	        this.Behavior = source["Behavior"];
	        this.Value = source["Value"];
	        this.Trigger = source["Trigger"];
	    }
	}
	export class PointerChainStep {
	    Offset: string;
	    ReadAddr: string;
	    PointerVal: string;
	
	    static createFrom(source: any = {}) {
	        return new PointerChainStep(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Offset = source["Offset"];
	        this.ReadAddr = source["ReadAddr"];
	        this.PointerVal = source["PointerVal"];
	    }
	}
	export class PointerChainResult {
	    StartAddr: string;
	    Steps: PointerChainStep[];
	    FinalAddr: string;
	    FinalInt32: number;
	    FinalFloat32: number;
	    Valid: boolean;
	    Err: string;
	
	    static createFrom(source: any = {}) {
	        return new PointerChainResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.StartAddr = source["StartAddr"];
	        this.Steps = this.convertValues(source["Steps"], PointerChainStep);
	        this.FinalAddr = source["FinalAddr"];
	        this.FinalInt32 = source["FinalInt32"];
	        this.FinalFloat32 = source["FinalFloat32"];
	        this.Valid = source["Valid"];
	        this.Err = source["Err"];
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
	
	export class PointerScanResult {
	    BaseOffset: string;
	    Offsets: string[];
	    Depth: number;
	
	    static createFrom(source: any = {}) {
	        return new PointerScanResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BaseOffset = source["BaseOffset"];
	        this.Offsets = source["Offsets"];
	        this.Depth = source["Depth"];
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
	
	export class Settings {
	    AutoConnect: boolean;
	    FreezeIntervalMs: number;
	    TrainerDir: string;
	    LaunchFullscreen: boolean;
	    WarnSystemProcesses: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.AutoConnect = source["AutoConnect"];
	        this.FreezeIntervalMs = source["FreezeIntervalMs"];
	        this.TrainerDir = source["TrainerDir"];
	        this.LaunchFullscreen = source["LaunchFullscreen"];
	        this.WarnSystemProcesses = source["WarnSystemProcesses"];
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

