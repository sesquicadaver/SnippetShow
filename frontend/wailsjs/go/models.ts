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
	    snippets: SnippetDTO[];
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
	        this.snippets = this.convertValues(source["snippets"], SnippetDTO);
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
	export class SnippetDTO {
	    id: number;
	    programId: number;
	    title: string;
	    command: string;
	    tags: string;
	    createdAt: string;

	    static createFrom(source: any = {}) {
	        return new SnippetDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.programId = source["programId"];
	        this.title = source["title"];
	        this.command = source["command"];
	        this.tags = source["tags"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class SnippetInput {
	    programId: number;
	    title: string;
	    command: string;
	    tags: string;

	    static createFrom(source: any = {}) {
	        return new SnippetInput(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.programId = source["programId"];
	        this.title = source["title"];
	        this.command = source["command"];
	        this.tags = source["tags"];
	    }
	}
	export class SafetyDTO {
	    dangerous: boolean;
	    reasons: string[];

	    static createFrom(source: any = {}) {
	        return new SafetyDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dangerous = source["dangerous"];
	        this.reasons = source["reasons"];
	    }
	}
	export class RunRequest {
	    command: string;
	    confirmed: boolean;

	    static createFrom(source: any = {}) {
	        return new RunRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.command = source["command"];
	        this.confirmed = source["confirmed"];
	    }
	}
	export class RunResponse {
	    started: boolean;
	    needsConfirm: boolean;
	    reasons: string[];
	    error: string;

	    static createFrom(source: any = {}) {
	        return new RunResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.started = source["started"];
	        this.needsConfirm = source["needsConfirm"];
	        this.reasons = source["reasons"];
	        this.error = source["error"];
	    }
	}
	export class CopyResponse {
	    ok: boolean;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new CopyResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.message = source["message"];
	    }
	}
	export class ScriptDraftDTO {
	    lines: string[];
	    count: number;
	    body: string;
	    shebang: string;

	    static createFrom(source: any = {}) {
	        return new ScriptDraftDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.lines = source["lines"];
	        this.count = source["count"];
	        this.body = source["body"];
	        this.shebang = source["shebang"];
	    }
	}
	export class CustomScriptDTO {
	    id: number;
	    name: string;
	    scriptBody: string;
	    updatedAt: string;

	    static createFrom(source: any = {}) {
	        return new CustomScriptDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.scriptBody = source["scriptBody"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class CustomScriptInput {
	    name: string;
	    scriptBody: string;

	    static createFrom(source: any = {}) {
	        return new CustomScriptInput(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.scriptBody = source["scriptBody"];
	    }
	}
	export class BackupResultDTO {
	    snippetsAdded: number;
	    snippetsSkipped: number;
	    scriptsAdded: number;
	    scriptsSkipped: number;
	    programsCreated: number;

	    static createFrom(source: any = {}) {
	        return new BackupResultDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.snippetsAdded = source["snippetsAdded"];
	        this.snippetsSkipped = source["snippetsSkipped"];
	        this.scriptsAdded = source["scriptsAdded"];
	        this.scriptsSkipped = source["scriptsSkipped"];
	        this.programsCreated = source["programsCreated"];
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
