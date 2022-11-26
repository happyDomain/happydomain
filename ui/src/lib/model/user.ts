import type UserSettings from './usersettings';

export interface User {
  id: string;
  email: string;
  CreatedAt: Date;
  LastSeen: Date;
  settings: UserSettings;
}

export interface SignUpForm {
    email: string;
    password: string;
    wantReceiveUpdate: boolean;
    lang: string;
}
