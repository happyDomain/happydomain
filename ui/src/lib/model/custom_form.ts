export class Field {
    id: string = "";
    type: string = "";
    label: string = "";
    placeholder: string = "";
    default: string = "";
    choices: Array<string> = [];
    required: boolean = false;
    secret: boolean = false;
    description: string = "";

    constructor(f: Field) {
        this.id = f.id;
        this.type = f.type;
        this.label = f.label;
        this.placeholder = f.placeholder;
        this.default = f.default;
        this.choices = f.choices;
        this.required = f.required;
        this.secret = f.secret;
        this.description = f.description;
    }
}

export class CustomForm {
    beforeText: string|null = null;
    sideText: string|null = null;
    afterText: string|null = null;
    fields: Array<Field> = [];
    nextButtonText: string|null = null;
    nextEditButtonText: string|null = null;
    previousButtonText: string|null = null;
    previousEditButtonText: string|null = null;
    nextButtonLink: string|null = null;
    nextButtonState: number|null = null;
    previousButtonLink: string|null = null;
    previousButtonState: number|null = null;

    constructor({beforeText, sideText, afterText, fields, nextButtonText, nextEditButtonText, previousButtonText, previousEditButtonText, nextButtonLink, nextButtonState, previousButtonLink, previousButtonState}: CustomForm) {
        this.beforeText = beforeText;
        this.sideText = sideText;
        this.afterText = afterText;
        this.fields = fields;
        this.nextButtonText = nextButtonText;
        this.nextEditButtonText = nextEditButtonText;
        this.previousButtonText = previousButtonText;
        this.previousEditButtonText = previousEditButtonText;
        this.nextButtonLink = nextButtonLink;
        this.nextButtonState = nextButtonState;
        this.previousButtonLink = previousButtonLink;
        this.previousButtonState = previousButtonState;
    }
}

export class FormState {
    _id?: any = null;
    _comment?: string = "";
    state: number = 0;
    recall: string|null = null;
    redirect?: string|null = null;

    constructor(o: FormState|null = null) {
        if (o) {
            const {_id, _comment, state, recall, redirect} = o;
            this._id = _id;
            this._comment = _comment;
            this.state = state;
            this.recall = recall;
            this.redirect = redirect;
        }
    }
}

export class FormResponse<T> {
    form: CustomForm | null = null;
    values: T | null = null;
    redirect: string | null = null;

    constructor(o: FormResponse<T> | null = null) {
        if (o) {
            const {form, values, redirect} = o;
            this.form = form;
            this.values = values;
            this.redirect = redirect;
        }
    }
}
