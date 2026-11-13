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

}

