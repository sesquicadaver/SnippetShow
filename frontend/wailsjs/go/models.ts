export namespace app {
	
	export class Info {
	    name: string;
	    subtitle: string;
	    module: string;
	    dbPath: string;
	    uiLocale: string;
	    phase: string;
	    frontend: string;
	    license: string;
	
	    static createFrom(source: any = {}) {
	        return new Info(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.subtitle = source["subtitle"];
	        this.module = source["module"];
	        this.dbPath = source["dbPath"];
	        this.uiLocale = source["uiLocale"];
	        this.phase = source["phase"];
	        this.frontend = source["frontend"];
	        this.license = source["license"];
	    }
	}
	export class ListQuery {
	    mode: string;
	    letter: string;
	    query: string;
	    limit: number;
	
	    static createFrom(source: any = {}) {
	        return new ListQuery(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.letter = source["letter"];
	        this.query = source["query"];
	        this.limit = source["limit"];
	    }
	}
	export class ManCommandDTO {
	    id: number;
	    flag: string;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new ManCommandDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.flag = source["flag"];
	        this.description = source["description"];
	    }
	}
	export class ProgramDTO {
	    id: number;
	    name: string;
	    packageName: string;
	    summary: string;
	    version: string;
	    isInstalled: boolean;
	    manParsed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ProgramDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.packageName = source["packageName"];
	        this.summary = source["summary"];
	        this.version = source["version"];
	        this.isInstalled = source["isInstalled"];
	        this.manParsed = source["manParsed"];
	    }
	}
	export class ProgramDetail {
	    program: ProgramDTO;
	    commands: ManCommandDTO[];
	    rawMan: string;
	    installCommand: string;
	    packageManager: string;
	    cacheHint: string;
	
	    static createFrom(source: any = {}) {
	        return new ProgramDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.program = this.convertValues(source["program"], ProgramDTO);
	        this.commands = this.convertValues(source["commands"], ManCommandDTO);
	        this.rawMan = source["rawMan"];
	        this.installCommand = source["installCommand"];
	        this.packageManager = source["packageManager"];
	        this.cacheHint = source["cacheHint"];
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
	export class ProgressEvent {
	    phase: string;
	    done: number;
	    total: number;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new ProgressEvent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.phase = source["phase"];
	        this.done = source["done"];
	        this.total = source["total"];
	        this.message = source["message"];
	    }
	}
	export class SearchResultDTO {
	    programId: number;
	    name: string;
	    summary: string;
	    rank: number;
	    isInstalled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SearchResultDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.programId = source["programId"];
	        this.name = source["name"];
	        this.summary = source["summary"];
	        this.rank = source["rank"];
	        this.isInstalled = source["isInstalled"];
	    }
	}

}

export namespace index {
	
	export class CatalogSyncResult {
	    added: number;
	    status: string;
	    hint: string;
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new CatalogSyncResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.added = source["added"];
	        this.status = source["status"];
	        this.hint = source["hint"];
	        this.total = source["total"];
	    }
	}
	export class UpdateResult {
	    added: number;
	    updated: number;
	    markedGone: number;
	    parsed: number;
	    installedSeen: number;
	
	    static createFrom(source: any = {}) {
	        return new UpdateResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.added = source["added"];
	        this.updated = source["updated"];
	        this.markedGone = source["markedGone"];
	        this.parsed = source["parsed"];
	        this.installedSeen = source["installedSeen"];
	    }
	}

}
