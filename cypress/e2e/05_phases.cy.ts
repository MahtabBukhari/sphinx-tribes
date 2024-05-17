import { User, HostName, UserStories, Phases } from '../support/objects/objects';

describe('Create Phases for Feature', () => {
    it('passes', () => {
        cy.upsertlogin(User).then(value => {
            for(let i = 0; i <= 2; i++) {
                cy.request({
                    method: 'POST',
                    url: `${HostName}/features/phase`,
                    headers: { 'x-jwt': `${value}` },
                    body: Phases[i]
                }).its('body').then(body => {
                    expect(body).to.have.property('uuid').and.equal(Phases[i].uuid.trim());
                    expect(body).to.have.property('feature_uuid').and.equal(Phases[i].feature_uuid.trim());
                    expect(body).to.have.property('name').and.equal(Phases[i].name.trim());
                    expect(body).to.have.property('priority').and.equal(Phases[i].priority);
                });
            }
        })
    })
})

describe('Modify phases name', () => {
    it('passes', () => {
        cy.upsertlogin(User).then(value => {
            for(let i = 0; i <= 2; i++) {
                cy.request({
                    method: 'POST',
                    url: `${HostName}/features/phase`,
                    headers: { 'x-jwt': `${value}` },
                    body: {
                        uuid: Phases[i].uuid,
                        name: Phases[i].name + "_addtext"
                    }
                }).its('body').then(body => {
                    expect(body).to.have.property('uuid').and.equal(Phases[i].uuid.trim());
                    expect(body).to.have.property('feature_uuid').and.equal(Phases[i].feature_uuid.trim());
                    expect(body).to.have.property('name').and.equal(Phases[i].name.trim() + " _addtext");
                    expect(body).to.have.property('priority').and.equal(Phases[i].priority);
                });
            }
        })
    })
})

describe('Get phases for feature', () => {
    it('passes', () => {
        cy.upsertlogin(User).then(value => {
            cy.request({
                method: 'GET',
                url: `${HostName}/features/${Phases[0].feature_uuid}/phase`,
                headers: { 'x-jwt': `${ value }` },
                body: {}
            }).then((resp) => {
                expect(resp.status).to.eq(200)

                resp.body.forEach((phase, index) => {
                    // Directly use index to compare with the expected phase in the same order
                    const expectedPhase = Phases[index];
                    expect(phase.uuid).to.equal(expectedPhase.uuid.trim());
                    expect(phase.feature_uuid).to.equal(expectedPhase.feature_uuid.trim());
                    expect(phase.name).to.equal(expectedPhase.name.trim() + " _addtext");
                    expect(phase.priority).to.equal(expectedPhase.priority);
                });
            })
        })
    })
})

describe('Get phase by uuid', () => {
    it('passes', () => {
        cy.upsertlogin(User).then(value => {
            for(let i = 0; i <= 2; i++) {
                cy.request({
                    method: 'GET',
                    url: `${HostName}/features/${Phases[0].feature_uuid}/phase/${Phases[i].uuid}`,
                    headers: { 'x-jwt': `${ value }` },
                    body: {}
                }).then((resp) => {
                    expect(resp.status).to.eq(200)
                    expect(resp.body).to.have.property('uuid').and.equal(Phases[i].uuid.trim());
                    expect(resp.body).to.have.property('feature_uuid').and.equal(Phases[i].feature_uuid.trim());
                    expect(resp.body).to.have.property('name').and.equal(Phases[i].name.trim() + " _addtext");
                    expect(resp.body).to.have.property('priority').and.equal(Phases[i].priority);
                })
            }
        })
    })
})

describe('Delete phase by uuid', () => {
    it('passes', () => {
        cy.upsertlogin(User).then(value => {
            cy.request({
                method: 'DELETE',
                url: `${HostName}/features/${Phases[0].feature_uuid}/phase/${Phases[0].uuid}`,
                headers: { 'x-jwt': `${ value }` },
                body: {}
            }).then((resp) => {
                expect(resp.status).to.eq(200)
            })
        })
    })
})

describe('Check delete by uuid', () => {
    it('passes', () => {
        cy.upsertlogin(User).then(value => {
            cy.request({
                method: 'GET',
                url: `${HostName}/features/${Phases[0].feature_uuid}/phase/${Phases[0].uuid}`,
                headers: { 'x-jwt': `${ value }` },
                body: {},
                failOnStatusCode: false
            }).then((resp) => {
                expect(resp.status).to.eq(404);
            })
        })
    })
})
