import type UserSettings from './usersettings';

export interface SignUpForm {
    email: string;
    password: string;
    wantReceiveUpdate: boolean;
    lang: string;
}

export interface LoginForm {
    email: string;
    password: string;
}

export class User {
    id: string;
    email: string;
    CreatedAt: Date;
    LastSeen: Date;
    settings: UserSettings;

    constructor({id, email, CreatedAt, LastSeen, settings}: User) {
        this.id = id;
        this.email = email;
        this.CreatedAt = CreatedAt;
        this.LastSeen = LastSeen;
        this.settings = settings;
    }
}

export default User;
