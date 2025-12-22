export namespace models {
	
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

