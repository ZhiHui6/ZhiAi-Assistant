export namespace festival {
	
	export class Day {
	    date: string;
	    label: string;
	    detail: string;
	    lunar: string;
	    kind: string;
	    holiday: boolean;
	    workday: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Day(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.label = source["label"];
	        this.detail = source["detail"];
	        this.lunar = source["lunar"];
	        this.kind = source["kind"];
	        this.holiday = source["holiday"];
	        this.workday = source["workday"];
	    }
	}

}

export namespace model {
	
	export class Alarm {
	    id: string;
	    time: string;
	    label?: string;
	    repeat: boolean;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Alarm(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.time = source["time"];
	        this.label = source["label"];
	        this.repeat = source["repeat"];
	        this.enabled = source["enabled"];
	    }
	}
	export class Countdown {
	    running: boolean;
	    endAt?: time.Time;
	    remaining: number;
	    label?: string;
	
	    static createFrom(source: any = {}) {
	        return new Countdown(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.endAt = this.convertValues(source["endAt"], time.Time);
	        this.remaining = source["remaining"];
	        this.label = source["label"];
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
	export class LaunchCategory {
	    id: string;
	    name: string;
	    order: number;
	    system?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new LaunchCategory(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.order = source["order"];
	        this.system = source["system"];
	    }
	}
	export class LaunchShortcut {
	    id: string;
	    categoryId: string;
	    path: string;
	    name: string;
	    baseName?: string;
	    order: number;
	    addedAt: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new LaunchShortcut(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.categoryId = source["categoryId"];
	        this.path = source["path"];
	        this.name = source["name"];
	        this.baseName = source["baseName"];
	        this.order = source["order"];
	        this.addedAt = this.convertValues(source["addedAt"], time.Time);
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
	export class LaunchData {
	    categories: LaunchCategory[];
	    items: LaunchShortcut[];
	
	    static createFrom(source: any = {}) {
	        return new LaunchData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.categories = this.convertValues(source["categories"], LaunchCategory);
	        this.items = this.convertValues(source["items"], LaunchShortcut);
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
	
	export class Mount {
	    id: string;
	    path: string;
	    label: string;
	    order: number;
	
	    static createFrom(source: any = {}) {
	        return new Mount(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.path = source["path"];
	        this.label = source["label"];
	        this.order = source["order"];
	    }
	}
	export class Note {
	    id: string;
	    text: string;
	    tags?: string[];
	    tint?: string;
	    pinned: boolean;
	    createdAt: time.Time;
	    updatedAt: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new Note(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.text = source["text"];
	        this.tags = source["tags"];
	        this.tint = source["tint"];
	        this.pinned = source["pinned"];
	        this.createdAt = this.convertValues(source["createdAt"], time.Time);
	        this.updatedAt = this.convertValues(source["updatedAt"], time.Time);
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
	export class Settings {
	    edge: string;
	    panelWidth: number;
	    peek: number;
	    targetMonitor: number;
	    hotkey: string;
	    language: string;
	    edgeEnabled: boolean;
	    autostart: boolean;
	    opacity: number;
	    background: string;
	    animations: boolean;
	    showClock: boolean;
	    showCalendar: boolean;
	    hideCalendarFestivals: boolean;
	    alarmSound: string;
	    countdownSound: string;
	    launchColumns: number;
	    launchRows: number;
	    launchIconSize: number;
	    hideLaunchNames: boolean;
	    hideLaunch: boolean;
	    clock12: boolean;
	    shelfView: string;
	    shelfScale: string;
	    mountView: string;
	    mountScale: string;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.edge = source["edge"];
	        this.panelWidth = source["panelWidth"];
	        this.peek = source["peek"];
	        this.targetMonitor = source["targetMonitor"];
	        this.hotkey = source["hotkey"];
	        this.language = source["language"];
	        this.edgeEnabled = source["edgeEnabled"];
	        this.autostart = source["autostart"];
	        this.opacity = source["opacity"];
	        this.background = source["background"];
	        this.animations = source["animations"];
	        this.showClock = source["showClock"];
	        this.showCalendar = source["showCalendar"];
	        this.hideCalendarFestivals = source["hideCalendarFestivals"];
	        this.alarmSound = source["alarmSound"];
	        this.countdownSound = source["countdownSound"];
	        this.launchColumns = source["launchColumns"];
	        this.launchRows = source["launchRows"];
	        this.launchIconSize = source["launchIconSize"];
	        this.hideLaunchNames = source["hideLaunchNames"];
	        this.hideLaunch = source["hideLaunch"];
	        this.clock12 = source["clock12"];
	        this.shelfView = source["shelfView"];
	        this.shelfScale = source["shelfScale"];
	        this.mountView = source["mountView"];
	        this.mountScale = source["mountScale"];
	    }
	}
	export class ShelfItem {
	    id: string;
	    groupId: string;
	    kind: string;
	    path: string;
	    name: string;
	    args?: string;
	    workDir?: string;
	    order: number;
	    addedAt: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new ShelfItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.groupId = source["groupId"];
	        this.kind = source["kind"];
	        this.path = source["path"];
	        this.name = source["name"];
	        this.args = source["args"];
	        this.workDir = source["workDir"];
	        this.order = source["order"];
	        this.addedAt = this.convertValues(source["addedAt"], time.Time);
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
	export class ShelfGroup {
	    id: string;
	    name: string;
	    order: number;
	    collapsed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ShelfGroup(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.order = source["order"];
	        this.collapsed = source["collapsed"];
	    }
	}
	export class ShelfData {
	    groups: ShelfGroup[];
	    items: ShelfItem[];
	
	    static createFrom(source: any = {}) {
	        return new ShelfData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.groups = this.convertValues(source["groups"], ShelfGroup);
	        this.items = this.convertValues(source["items"], ShelfItem);
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
	
	
	export class ShutdownPlan {
	    active: boolean;
	    at?: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new ShutdownPlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.active = source["active"];
	        this.at = this.convertValues(source["at"], time.Time);
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
	export class TimerData {
	    alarms: Alarm[];
	    countdown: Countdown;
	    shutdown: ShutdownPlan;
	
	    static createFrom(source: any = {}) {
	        return new TimerData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.alarms = this.convertValues(source["alarms"], Alarm);
	        this.countdown = this.convertValues(source["countdown"], Countdown);
	        this.shutdown = this.convertValues(source["shutdown"], ShutdownPlan);
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
	export class Todo {
	    id: string;
	    title: string;
	    note?: string;
	    done: boolean;
	    priority: number;
	    dueDate?: string;
	    group?: string;
	    order: number;
	    createdAt: time.Time;
	    completedAt?: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new Todo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.note = source["note"];
	        this.done = source["done"];
	        this.priority = source["priority"];
	        this.dueDate = source["dueDate"];
	        this.group = source["group"];
	        this.order = source["order"];
	        this.createdAt = this.convertValues(source["createdAt"], time.Time);
	        this.completedAt = this.convertValues(source["completedAt"], time.Time);
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
	export class UsagePack {
	    name?: string;
	    total: number;
	    remaining: number;
	    unit?: string;
	    expiresAt?: string;
	    url?: string;
	
	    static createFrom(source: any = {}) {
	        return new UsagePack(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.total = source["total"];
	        this.remaining = source["remaining"];
	        this.unit = source["unit"];
	        this.expiresAt = source["expiresAt"];
	        this.url = source["url"];
	    }
	}
	export class UsageSnapshot {
	    account?: string;
	    plan?: string;
	    unit?: string;
	    total: number;
	    used: number;
	    remaining: number;
	    percent: number;
	    resetAt?: string;
	    dailyCredit?: number;
	    checkedIn: boolean;
	    checkinAvailable: boolean;
	    message?: string;
	    packs?: UsagePack[];
	    updatedAt: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new UsageSnapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.account = source["account"];
	        this.plan = source["plan"];
	        this.unit = source["unit"];
	        this.total = source["total"];
	        this.used = source["used"];
	        this.remaining = source["remaining"];
	        this.percent = source["percent"];
	        this.resetAt = source["resetAt"];
	        this.dailyCredit = source["dailyCredit"];
	        this.checkedIn = source["checkedIn"];
	        this.checkinAvailable = source["checkinAvailable"];
	        this.message = source["message"];
	        this.packs = this.convertValues(source["packs"], UsagePack);
	        this.updatedAt = this.convertValues(source["updatedAt"], time.Time);
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
	export class UsagePlatformState {
	    snapshot: UsageSnapshot;
	    source: string;
	    authState: string;
	    lastError?: string;
	    lastFetchAt?: time.Time;
	    lastCheckinDate?: string;
	    attemptDate?: string;
	    retryAt?: time.Time;
	    attempts?: number;
	    signedOut?: boolean;
	    preferred?: string;
	
	    static createFrom(source: any = {}) {
	        return new UsagePlatformState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.snapshot = this.convertValues(source["snapshot"], UsageSnapshot);
	        this.source = source["source"];
	        this.authState = source["authState"];
	        this.lastError = source["lastError"];
	        this.lastFetchAt = this.convertValues(source["lastFetchAt"], time.Time);
	        this.lastCheckinDate = source["lastCheckinDate"];
	        this.attemptDate = source["attemptDate"];
	        this.retryAt = this.convertValues(source["retryAt"], time.Time);
	        this.attempts = source["attempts"];
	        this.signedOut = source["signedOut"];
	        this.preferred = source["preferred"];
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
	
	export class UsageState {
	    platforms: Record<string, UsagePlatformState>;
	    autoCheckin: Record<string, boolean>;
	    noticeAcked: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UsageState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.platforms = this.convertValues(source["platforms"], UsagePlatformState, true);
	        this.autoCheckin = source["autoCheckin"];
	        this.noticeAcked = source["noticeAcked"];
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

export namespace mount {
	
	export class Entry {
	    name: string;
	    path: string;
	    isDir: boolean;
	    ext: string;
	    size: number;
	    modTime: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new Entry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.isDir = source["isDir"];
	        this.ext = source["ext"];
	        this.size = source["size"];
	        this.modTime = this.convertValues(source["modTime"], time.Time);
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
	export class ListOptions {
	    showHidden: boolean;
	    sortBy: string;
	    desc: boolean;
	    limit: number;
	    offset: number;
	
	    static createFrom(source: any = {}) {
	        return new ListOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.showHidden = source["showHidden"];
	        this.sortBy = source["sortBy"];
	        this.desc = source["desc"];
	        this.limit = source["limit"];
	        this.offset = source["offset"];
	    }
	}

}

export namespace service {
	
	export class AboutInfo {
	    name: string;
	    version: string;
	    arch: string;
	    dataDir: string;
	    projectUrl: string;
	    releasesUrl: string;
	    issuesUrl: string;
	
	    static createFrom(source: any = {}) {
	        return new AboutInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.version = source["version"];
	        this.arch = source["arch"];
	        this.dataDir = source["dataDir"];
	        this.projectUrl = source["projectUrl"];
	        this.releasesUrl = source["releasesUrl"];
	        this.issuesUrl = source["issuesUrl"];
	    }
	}
	export class BackupCategoryResult {
	    id: string;
	    files: number;
	    bytes: number;
	
	    static createFrom(source: any = {}) {
	        return new BackupCategoryResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.files = source["files"];
	        this.bytes = source["bytes"];
	    }
	}
	export class BackupContentsResult {
	    categories: BackupCategoryResult[];
	    legacy: boolean;
	
	    static createFrom(source: any = {}) {
	        return new BackupContentsResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.categories = this.convertValues(source["categories"], BackupCategoryResult);
	        this.legacy = source["legacy"];
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
	export class BackupResult {
	    path: string;
	    bytes: number;
	    files: number;
	
	    static createFrom(source: any = {}) {
	        return new BackupResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.bytes = source["bytes"];
	        this.files = source["files"];
	    }
	}
	export class Calendar {
	
	
	    static createFrom(source: any = {}) {
	        return new Calendar(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class DataLocationInfo {
	    mode: string;
	    custom: string;
	    effective: string;
	    defaultDir: string;
	    portableDir: string;
	    fellBack: boolean;
	    bytes: number;
	
	    static createFrom(source: any = {}) {
	        return new DataLocationInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.custom = source["custom"];
	        this.effective = source["effective"];
	        this.defaultDir = source["defaultDir"];
	        this.portableDir = source["portableDir"];
	        this.fellBack = source["fellBack"];
	        this.bytes = source["bytes"];
	    }
	}
	export class Icons {
	
	
	    static createFrom(source: any = {}) {
	        return new Icons(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class Launch {
	
	
	    static createFrom(source: any = {}) {
	        return new Launch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class Mount {
	
	
	    static createFrom(source: any = {}) {
	        return new Mount(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class Notes {
	
	
	    static createFrom(source: any = {}) {
	        return new Notes(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class Shelf {
	
	
	    static createFrom(source: any = {}) {
	        return new Shelf(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class Thumbs {
	
	
	    static createFrom(source: any = {}) {
	        return new Thumbs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class Timers {
	
	
	    static createFrom(source: any = {}) {
	        return new Timers(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class Todos {
	
	
	    static createFrom(source: any = {}) {
	        return new Todos(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class UpdateInfo {
	    current: string;
	    latest: string;
	    hasUpdate: boolean;
	    canInstall: boolean;
	    notes: string;
	    pageUrl: string;
	    downloadUrl: string;
	    checksumUrl: string;
	    assetName: string;
	    sha256: string;
	    size: number;
	    publishedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.current = source["current"];
	        this.latest = source["latest"];
	        this.hasUpdate = source["hasUpdate"];
	        this.canInstall = source["canInstall"];
	        this.notes = source["notes"];
	        this.pageUrl = source["pageUrl"];
	        this.downloadUrl = source["downloadUrl"];
	        this.checksumUrl = source["checksumUrl"];
	        this.assetName = source["assetName"];
	        this.sha256 = source["sha256"];
	        this.size = source["size"];
	        this.publishedAt = source["publishedAt"];
	    }
	}
	export class Usage {
	
	
	    static createFrom(source: any = {}) {
	        return new Usage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class Window {
	
	
	    static createFrom(source: any = {}) {
	        return new Window(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class WindowInfo {
	    state: string;
	    pinned: boolean;
	    edge: string;
	    edgeEnabled: boolean;
	    panelW: number;
	    panelH: number;
	    x: number;
	    y: number;
	    dpi: number;
	    monitor: number;
	    hotkey: string;
	    hotkeyOk: boolean;
	
	    static createFrom(source: any = {}) {
	        return new WindowInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.state = source["state"];
	        this.pinned = source["pinned"];
	        this.edge = source["edge"];
	        this.edgeEnabled = source["edgeEnabled"];
	        this.panelW = source["panelW"];
	        this.panelH = source["panelH"];
	        this.x = source["x"];
	        this.y = source["y"];
	        this.dpi = source["dpi"];
	        this.monitor = source["monitor"];
	        this.hotkey = source["hotkey"];
	        this.hotkeyOk = source["hotkeyOk"];
	    }
	}

}

export namespace time {
	
	export class Time {
	
	
	    static createFrom(source: any = {}) {
	        return new Time(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}

}

export namespace todos {
	
	export class DaySummary {
	    date: string;
	    open: number;
	    done: number;
	    overdue: number;
	
	    static createFrom(source: any = {}) {
	        return new DaySummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.open = source["open"];
	        this.done = source["done"];
	        this.overdue = source["overdue"];
	    }
	}
	export class Filter {
	    scope: string;
	    date?: string;
	    limit?: number;
	
	    static createFrom(source: any = {}) {
	        return new Filter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.scope = source["scope"];
	        this.date = source["date"];
	        this.limit = source["limit"];
	    }
	}
	export class Input {
	    title: string;
	    note?: string;
	    priority: number;
	    dueDate?: string;
	    group?: string;
	
	    static createFrom(source: any = {}) {
	        return new Input(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.note = source["note"];
	        this.priority = source["priority"];
	        this.dueDate = source["dueDate"];
	        this.group = source["group"];
	    }
	}
	export class Patch {
	    title?: string;
	    note?: string;
	    priority?: number;
	    dueDate?: string;
	    clearDue?: boolean;
	    group?: string;
	    done?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Patch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.note = source["note"];
	        this.priority = source["priority"];
	        this.dueDate = source["dueDate"];
	        this.clearDue = source["clearDue"];
	        this.group = source["group"];
	        this.done = source["done"];
	    }
	}
	export class Stats {
	    today: number;
	    todayDone: number;
	    week: number;
	    overdue: number;
	    open: number;
	    done: number;
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new Stats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.today = source["today"];
	        this.todayDone = source["todayDone"];
	        this.week = source["week"];
	        this.overdue = source["overdue"];
	        this.open = source["open"];
	        this.done = source["done"];
	        this.total = source["total"];
	    }
	}

}

export namespace usage {
	
	export class BrowserLoginPoll {
	    done: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new BrowserLoginPoll(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.done = source["done"];
	        this.error = source["error"];
	    }
	}
	export class BrowserLoginStart {
	    id: string;
	    url: string;
	    expiresIn: number;
	
	    static createFrom(source: any = {}) {
	        return new BrowserLoginStart(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.url = source["url"];
	        this.expiresIn = source["expiresIn"];
	    }
	}
	export class LoginPoll {
	    done: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new LoginPoll(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.done = source["done"];
	        this.error = source["error"];
	    }
	}
	export class LoginStart {
	    id: string;
	    url: string;
	    expiresIn: number;
	
	    static createFrom(source: any = {}) {
	        return new LoginStart(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.url = source["url"];
	        this.expiresIn = source["expiresIn"];
	    }
	}

}

