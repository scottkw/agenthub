export namespace daemon {
	
	export class DetectedShell {
	    name: string;
	    displayName: string;
	    path: string;
	    argv: string[];
	
	    static createFrom(source: any = {}) {
	        return new DetectedShell(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.displayName = source["displayName"];
	        this.path = source["path"];
	        this.argv = source["argv"];
	    }
	}
	export class ImageConfig {
	    storageLimit: number;
	
	    static createFrom(source: any = {}) {
	        return new ImageConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.storageLimit = source["storageLimit"];
	    }
	}
	export class IssueCapabilitiesResponse {
	    readUrl: string;
	    writeUrl: string;
	    readCode: string;
	    writeCode: string;
	    homeDir: boolean;
	
	    static createFrom(source: any = {}) {
	        return new IssueCapabilitiesResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.readUrl = source["readUrl"];
	        this.writeUrl = source["writeUrl"];
	        this.readCode = source["readCode"];
	        this.writeCode = source["writeCode"];
	        this.homeDir = source["homeDir"];
	    }
	}
	export class WebLinksConfig {
	    modifier: string;
	    confirmOSC8: boolean;
	    confirmIDN: boolean;
	    confirmTyposquat: boolean;
	
	    static createFrom(source: any = {}) {
	        return new WebLinksConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.modifier = source["modifier"];
	        this.confirmOSC8 = source["confirmOSC8"];
	        this.confirmIDN = source["confirmIDN"];
	        this.confirmTyposquat = source["confirmTyposquat"];
	    }
	}
	export class SearchConfig {
	    regex: boolean;
	    caseSensitive: boolean;
	    wholeWord: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SearchConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.regex = source["regex"];
	        this.caseSensitive = source["caseSensitive"];
	        this.wholeWord = source["wholeWord"];
	    }
	}
	export class PluginSettings {
	    webgl: boolean;
	    unicode11: boolean;
	    search: boolean;
	    searchConfig: SearchConfig;
	    webLinks: boolean;
	    webLinksConfig: WebLinksConfig;
	    image: boolean;
	    imageConfig: ImageConfig;
	    serialize: boolean;
	    clipboard: boolean;
	    progress: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PluginSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.webgl = source["webgl"];
	        this.unicode11 = source["unicode11"];
	        this.search = source["search"];
	        this.searchConfig = this.convertValues(source["searchConfig"], SearchConfig);
	        this.webLinks = source["webLinks"];
	        this.webLinksConfig = this.convertValues(source["webLinksConfig"], WebLinksConfig);
	        this.image = source["image"];
	        this.imageConfig = this.convertValues(source["imageConfig"], ImageConfig);
	        this.serialize = source["serialize"];
	        this.clipboard = source["clipboard"];
	        this.progress = source["progress"];
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
	

}

export namespace main {
	
	export class RemoteSession {
	    id: string;
	    name: string;
	    cliType: string;
	    status: string;
	    url: string;
	
	    static createFrom(source: any = {}) {
	        return new RemoteSession(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.cliType = source["cliType"];
	        this.status = source["status"];
	        this.url = source["url"];
	    }
	}
	export class RemotePeerSessions {
	    hostname: string;
	    reachable: boolean;
	    sessions: RemoteSession[];
	
	    static createFrom(source: any = {}) {
	        return new RemotePeerSessions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hostname = source["hostname"];
	        this.reachable = source["reachable"];
	        this.sessions = this.convertValues(source["sessions"], RemoteSession);
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
	
	export class SessionInfo {
	    id: string;
	    cli: string;
	    name: string;
	    state: string;
	    status: string;
	    createdAt: string;
	    hostname: string;
	    webEnabled: boolean;
	    homeDir: boolean;
	    filesWrite: boolean;
	    viewerCount: number;
	    exitCode?: number;
	    duration?: number;
	    workDir: string;
	
	    static createFrom(source: any = {}) {
	        return new SessionInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.cli = source["cli"];
	        this.name = source["name"];
	        this.state = source["state"];
	        this.status = source["status"];
	        this.createdAt = source["createdAt"];
	        this.hostname = source["hostname"];
	        this.webEnabled = source["webEnabled"];
	        this.homeDir = source["homeDir"];
	        this.filesWrite = source["filesWrite"];
	        this.viewerCount = source["viewerCount"];
	        this.exitCode = source["exitCode"];
	        this.duration = source["duration"];
	        this.workDir = source["workDir"];
	    }
	}

}

export namespace pty {
	
	export class DetectedCLI {
	    Name: string;
	    DisplayName: string;
	    Path: string;
	
	    static createFrom(source: any = {}) {
	        return new DetectedCLI(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.DisplayName = source["DisplayName"];
	        this.Path = source["Path"];
	    }
	}

}

export namespace updater {
	
	export class UpdateInfo {
	    currentVersion: string;
	    latestVersion: string;
	    releaseURL: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.currentVersion = source["currentVersion"];
	        this.latestVersion = source["latestVersion"];
	        this.releaseURL = source["releaseURL"];
	    }
	}

}

export namespace webserver {
	
	export class TailscaleHealth {
	    installed: boolean;
	    connected: boolean;
	    hasCerts: boolean;
	    ip: string;
	    domain: string;
	    binaryFound: boolean;
	    daemonUp: boolean;
	    platformHint: string;
	    acceptDns: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TailscaleHealth(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.installed = source["installed"];
	        this.connected = source["connected"];
	        this.hasCerts = source["hasCerts"];
	        this.ip = source["ip"];
	        this.domain = source["domain"];
	        this.binaryFound = source["binaryFound"];
	        this.daemonUp = source["daemonUp"];
	        this.platformHint = source["platformHint"];
	        this.acceptDns = source["acceptDns"];
	    }
	}

}

