<script lang="ts">
 import '../app.scss'
 import "bootstrap-icons/font/bootstrap-icons.css";

 import {
     Col,
     Container,
     Row,
     //Styles,
 } from 'sveltestrap';

 import Header from '$lib/components/Header.svelte';
 import Logo from '$lib/components/Logo.svelte';
 import Toaster from '$lib/components/Toaster.svelte';
 import VoxPeople from '$lib/components/VoxPeople.svelte';
</script>

<svelte:head>
    <title>happyDomain</title>
</svelte:head>

<!--Styles /-->

<Header />

<div class="flex-fill d-flex flex-column justify-content-center">
    <slot></slot>
</div>

<Toaster />
<VoxPeople />

<footer class="pt-2 pb-2 bg-dark text-light">
    <Container>
        <Row>
            <Col md="12" lg="6">
                &copy;
                <Logo color="#fff" height="17" />
                2019-2022 All rights reserved
            </Col>
            <Col md="6" lg="3" />
            <Col md="6" lg="3" />
        </Row>
    </Container>
</footer>
