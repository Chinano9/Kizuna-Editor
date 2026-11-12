export namespace audio {
	
	export class AudioDevice {
	    id: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new AudioDevice(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	    }
	}

}

export namespace bindings {
	
	export class AlbumHandler {
	
	
	    static createFrom(source: any = {}) {
	        return new AlbumHandler(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class AudioHandler {
	
	
	    static createFrom(source: any = {}) {
	        return new AudioHandler(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class SongHandler {
	
	
	    static createFrom(source: any = {}) {
	        return new SongHandler(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class SyncHandler {
	
	
	    static createFrom(source: any = {}) {
	        return new SyncHandler(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class TrackHandler {
	
	
	    static createFrom(source: any = {}) {
	        return new TrackHandler(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}

}

export namespace models {
	
	export class Album {
	    id: number;
	    title: string;
	    artist: string;
	    description: string;
	    // Go type: time
	    created_at: any;
	
	    static createFrom(source: any = {}) {
	        return new Album(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.artist = source["artist"];
	        this.description = source["description"];
	        this.created_at = this.convertValues(source["created_at"], null);
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
	export class AudioVersion {
	    id: number;
	    song_id: number;
	    track_id?: number;
	    version_name: string;
	    file_path: string;
	    notes: string;
	    // Go type: time
	    created_at: any;
	
	    static createFrom(source: any = {}) {
	        return new AudioVersion(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.song_id = source["song_id"];
	        this.track_id = source["track_id"];
	        this.version_name = source["version_name"];
	        this.file_path = source["file_path"];
	        this.notes = source["notes"];
	        this.created_at = this.convertValues(source["created_at"], null);
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
	export class CloudPushResult {
	    project_id: number;
	    new_version: number;
	    // Go type: time
	    synced_at: any;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new CloudPushResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.project_id = source["project_id"];
	        this.new_version = source["new_version"];
	        this.synced_at = this.convertValues(source["synced_at"], null);
	        this.status = source["status"];
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
	export class Instrument {
	    id: number;
	    name: string;
	    type: string;
	    default_clef: string;
	
	    static createFrom(source: any = {}) {
	        return new Instrument(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.default_clef = source["default_clef"];
	    }
	}
	export class QuickIdea {
	    id: number;
	    file_path: string;
	    // Go type: time
	    created_at: any;
	    duration_seconds?: number;
	
	    static createFrom(source: any = {}) {
	        return new QuickIdea(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.file_path = source["file_path"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.duration_seconds = source["duration_seconds"];
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
	export class Track {
	    id: number;
	    song_id: number;
	    instrument_id: number;
	    name: string;
	    data_content: string;
	    display_mode: string;
	    is_muted: boolean;
	    // Go type: time
	    created_at: any;
	
	    static createFrom(source: any = {}) {
	        return new Track(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.song_id = source["song_id"];
	        this.instrument_id = source["instrument_id"];
	        this.name = source["name"];
	        this.data_content = source["data_content"];
	        this.display_mode = source["display_mode"];
	        this.is_muted = source["is_muted"];
	        this.created_at = this.convertValues(source["created_at"], null);
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
	export class Song {
	    id: number;
	    album_id?: number;
	    title: string;
	    bpm: number;
	    time_signature: string;
	    key_signature: string;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	    tracks?: Track[];
	
	    static createFrom(source: any = {}) {
	        return new Song(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.album_id = source["album_id"];
	        this.title = source["title"];
	        this.bpm = source["bpm"];
	        this.time_signature = source["time_signature"];
	        this.key_signature = source["key_signature"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	        this.tracks = this.convertValues(source["tracks"], Track);
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

