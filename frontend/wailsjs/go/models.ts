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
	    record_victim_view: boolean;
	    play_team_voice: boolean;
	    killer_pre_seconds: number;
	    killer_post_seconds: number;
	    victim_pre_seconds: number;
	    victim_post_seconds: number;
	
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
	        this.record_victim_view = source["record_victim_view"];
	        this.play_team_voice = source["play_team_voice"];
	        this.killer_pre_seconds = source["killer_pre_seconds"];
	        this.killer_post_seconds = source["killer_post_seconds"];
	        this.victim_pre_seconds = source["victim_pre_seconds"];
	        this.victim_post_seconds = source["victim_post_seconds"];
	    }
	}
	export class CountResponse {
	    counts: number;
	
	    static createFrom(source: any = {}) {
	        return new CountResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.counts = source["counts"];
	    }
	}
	export class KillInfo {
	    round: number;
	    tick: number;
	    map_name: string;
	    victim_name: string;
	    victim_entity_id: number;
	    victim_steam_id: number;
	    victim_side: string;
	    victim_x: number;
	    victim_y: number;
	    victim_z: number;
	    killer_name: string;
	    killer_steam_id: number;
	    killer_side: string;
	    killer_x: number;
	    killer_y: number;
	    killer_z: number;
	    weapon_name: string;
	    is_headshot: boolean;
	    is_wallbang: boolean;
	    can_render_2d_kill: boolean;
	
	    static createFrom(source: any = {}) {
	        return new KillInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.round = source["round"];
	        this.tick = source["tick"];
	        this.map_name = source["map_name"];
	        this.victim_name = source["victim_name"];
	        this.victim_entity_id = source["victim_entity_id"];
	        this.victim_steam_id = source["victim_steam_id"];
	        this.victim_side = source["victim_side"];
	        this.victim_x = source["victim_x"];
	        this.victim_y = source["victim_y"];
	        this.victim_z = source["victim_z"];
	        this.killer_name = source["killer_name"];
	        this.killer_steam_id = source["killer_steam_id"];
	        this.killer_side = source["killer_side"];
	        this.killer_x = source["killer_x"];
	        this.killer_y = source["killer_y"];
	        this.killer_z = source["killer_z"];
	        this.weapon_name = source["weapon_name"];
	        this.is_headshot = source["is_headshot"];
	        this.is_wallbang = source["is_wallbang"];
	        this.can_render_2d_kill = source["can_render_2d_kill"];
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
	
	export class Map2DRenderData {
	    map_name: string;
	    pos_x: number;
	    pos_y: number;
	    scale: number;
	    image_data: string;
	
	    static createFrom(source: any = {}) {
	        return new Map2DRenderData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.map_name = source["map_name"];
	        this.pos_x = source["pos_x"];
	        this.pos_y = source["pos_y"];
	        this.scale = source["scale"];
	        this.image_data = source["image_data"];
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
	export class UsageStats {
	    run: number;
	    make: number;
	
	    static createFrom(source: any = {}) {
	        return new UsageStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.run = source["run"];
	        this.make = source["make"];
	    }
	}

}

