export namespace main {
	
	export class Config {
	    cs2_exe: string;
	    hlae_exe: string;
	    hlae_version: string;
	    ffmpeg_dir: string;
	    cfg_dir: string;
	    output_dir: string;
	    record_fps: number;
	    tickrate: number;
	    video_preset: string;
	    transition_duration: number;
	    transition_type: string;
	    launch_resolution: string;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cs2_exe = source["cs2_exe"];
	        this.hlae_exe = source["hlae_exe"];
	        this.hlae_version = source["hlae_version"];
	        this.ffmpeg_dir = source["ffmpeg_dir"];
	        this.cfg_dir = source["cfg_dir"];
	        this.output_dir = source["output_dir"];
	        this.record_fps = source["record_fps"];
	        this.tickrate = source["tickrate"];
	        this.video_preset = source["video_preset"];
	        this.transition_duration = source["transition_duration"];
	        this.transition_type = source["transition_type"];
	        this.launch_resolution = source["launch_resolution"];
	    }
	}
	export class KillInfo {
	    round: number;
	    tick: number;
	    victim_name: string;
	    killer_name: string;
	    weapon_name: string;
	    is_headshot: boolean;
	    is_wallbang: boolean;
	
	    static createFrom(source: any = {}) {
	        return new KillInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.round = source["round"];
	        this.tick = source["tick"];
	        this.victim_name = source["victim_name"];
	        this.killer_name = source["killer_name"];
	        this.weapon_name = source["weapon_name"];
	        this.is_headshot = source["is_headshot"];
	        this.is_wallbang = source["is_wallbang"];
	    }
	}
	export class RoundSummary {
	    round: number;
	    kills: KillInfo[];
	
	    static createFrom(source: any = {}) {
	        return new RoundSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.round = source["round"];
	        this.kills = this.convertValues(source["kills"], KillInfo);
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
	export class PlayerSummary {
	    name: string;
	    steam_id: string;
	    entity_id: number;
	    total_kills: number;
	    rounds: RoundSummary[];
	
	    static createFrom(source: any = {}) {
	        return new PlayerSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.steam_id = source["steam_id"];
	        this.entity_id = source["entity_id"];
	        this.total_kills = source["total_kills"];
	        this.rounds = this.convertValues(source["rounds"], RoundSummary);
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
	export class DemoInfo {
	    demo_path: string;
	    players: PlayerSummary[];
	
	    static createFrom(source: any = {}) {
	        return new DemoInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.demo_path = source["demo_path"];
	        this.players = this.convertValues(source["players"], PlayerSummary);
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
	
	
	export class RecordRequest {
	    demo_path: string;
	    player_steam_id: string;
	    selected_rounds: number[];
	    auto_mode: boolean;
	    debug_mode: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RecordRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.demo_path = source["demo_path"];
	        this.player_steam_id = source["player_steam_id"];
	        this.selected_rounds = source["selected_rounds"];
	        this.auto_mode = source["auto_mode"];
	        this.debug_mode = source["debug_mode"];
	    }
	}
	export class RecordResult {
	    cfg_path: string;
	    output_path: string;
	
	    static createFrom(source: any = {}) {
	        return new RecordResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cfg_path = source["cfg_path"];
	        this.output_path = source["output_path"];
	    }
	}
	
	export class UpdateInfo {
	    available: boolean;
	    current: string;
	    latest: string;
	    url: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.current = source["current"];
	        this.latest = source["latest"];
	        this.url = source["url"];
	    }
	}

}

