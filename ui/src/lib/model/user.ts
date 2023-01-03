import { UserSettings } from './usersettings';

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
    id: string = "";
    email: string = "";
    CreatedAt: Date = new Date("0000-00-00T00:00:00");
    LastSeen: Date = new Date("0000-00-00T00:00:00");
    settings: UserSettings = new UserSettings();
}

export default User;
