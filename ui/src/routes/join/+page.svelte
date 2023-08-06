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
        <Col sm="4" class="d-none d-sm-flex flex-column justify-content-center">
            <img src="img/signup.webp" alt="Welcome aboard!">
        </Col>
        <Col sm="8" class="d-flex flex-column justify-content-center">
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
            <div class="mt-3 text-justify">
                Join now our open source and free (as freedom) DNS platform, to manage your domains easily!
            </div>
        </Col>
    </Row>
</Container>
