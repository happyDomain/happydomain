<script lang="ts">
 import {
     Card,
     CardBody,
     CardHeader,
     Container,
     Col,
     Row,
 } from 'sveltestrap';

 import { t } from '$lib/translations';
 import SignUpForm from '$lib/components/SignUpForm.svelte';
</script>

<Container class="my-3">
    <Row>
        <Col sm="4" class="d-none d-sm-flex flex-column align-items-center">
            <img src="img/screenshot.png" alt="HappyDNS screenshoot" style="max-height: 100%; max-width: 100%;" class="mt-auto">
            <div class="mt-3 mb-auto text-justify">
                Join now our open source and free (as freedom) DNS platform, to manage your domains easily!
            </div>
        </Col>
        <Col sm="8">
            <Card>
                <CardHeader>
                    <h6 class="card-title my-1 fw-bold">
                        {$t('account.signup.join-call')}
                    </h6>
                </CardHeader>
                <CardBody>
                    <SignUpForm />
                </CardBody>
            </Card>
        </Col>
    </Row>
</Container>
