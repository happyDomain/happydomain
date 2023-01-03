export const FieldHintHide = 0;
export const FieldHintTooltip = 1;
export const FieldHintFocused = 2;
export const FieldHintAlways = 3;

export const ZoneViewGrid = 0;
export const ZoneViewList = 1;
export const ZoneViewRecords = 2;

export class UserSettings {
    language: string = "";
    newsletter: boolean = false;
    fieldhint: number = FieldHintFocused;
    zoneview: number = ZoneViewGrid;
}
