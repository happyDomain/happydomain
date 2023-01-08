export interface Field {
    id: string;
    type: string;
    label?: string;
    placeholder?: string;
    default?: string;
    choices?: Array<string>;
    required?: boolean;
    secret?: boolean;
    description?: string;
}

export interface CustomForm {
    beforeText?: string;
    sideText?: string;
    afterText?: string;
    fields: Array<Field>;
    nextButtonText?: string;
    nextEditButtonText?: string;
    previousButtonText?: string;
    previousEditButtonText?: string;
    nextButtonLink?: string;
    nextButtonState?: number;
    previousButtonLink?: string;
    previousButtonState?: number;
}

export interface FormState {
    _id?: any;
    _comment?: string;
    state: number;
    recall?: string;
    redirect?: string;
}

export interface FormResponse<T> {
    form?: CustomForm;
    values?: T;
    redirect?: string;
}
