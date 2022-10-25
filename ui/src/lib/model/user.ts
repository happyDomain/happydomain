import type UserSettings from './usersettings';

interface User {
  id: string;
  email: string;
  CreatedAt: Date;
  LastSeen: Date;
  settings: UserSettings;
}

export default User;
