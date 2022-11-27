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
 import LoginForm from '$lib/components/LoginForm.svelte';
</script>

<Container class="my-3">
    <Row>
        <Col sm="4" class="d-none d-sm-flex flex-column justify-content-center">
          <div>
              {$t('account.ask-have')}
              <a href="/join" class="fw-bold">
                  {$t('account.join')}
              </a>
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
                    <LoginForm />
                </CardBody>
            </Card>
        </Col>
    </Row>
</Container>
