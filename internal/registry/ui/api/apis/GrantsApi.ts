/* tslint:disable */
/* eslint-disable */
/**
 * Infra API
 * Infra REST API
 *
 * The version of the OpenAPI document: 0.1.0
 * 
 *
 * NOTE: This class is auto generated by OpenAPI Generator (https://openapi-generator.tech).
 * https://openapi-generator.tech
 * Do not edit the class manually.
 */


import * as runtime from '../runtime';
import {
    Grant,
    GrantFromJSON,
    GrantToJSON,
    GrantKind,
    GrantKindFromJSON,
    GrantKindToJSON,
} from '../models';

export interface GetGrantRequest {
    id: string;
}

export interface ListGrantsRequest {
    name?: string;
    kind?: GrantKind;
    destination?: string;
}

/**
 * 
 */
export class GrantsApi extends runtime.BaseAPI {

    /**
     * Get grant
     */
    async getGrantRaw(requestParameters: GetGrantRequest, initOverrides?: RequestInit): Promise<runtime.ApiResponse<Grant>> {
        if (requestParameters.id === null || requestParameters.id === undefined) {
            throw new runtime.RequiredError('id','Required parameter requestParameters.id was null or undefined when calling getGrant.');
        }

        const queryParameters: any = {};

        const headerParameters: runtime.HTTPHeaders = {};

        if (this.configuration && this.configuration.accessToken) {
            const token = this.configuration.accessToken;
            const tokenString = await token("bearerAuth", []);

            if (tokenString) {
                headerParameters["Authorization"] = `Bearer ${tokenString}`;
            }
        }
        const response = await this.request({
            path: `/grants/{id}`.replace(`{${"id"}}`, encodeURIComponent(String(requestParameters.id))),
            method: 'GET',
            headers: headerParameters,
            query: queryParameters,
        }, initOverrides);

        return new runtime.JSONApiResponse(response, (jsonValue) => GrantFromJSON(jsonValue));
    }

    /**
     * Get grant
     */
    async getGrant(requestParameters: GetGrantRequest, initOverrides?: RequestInit): Promise<Grant> {
        const response = await this.getGrantRaw(requestParameters, initOverrides);
        return await response.value();
    }

    /**
     * List grants
     */
    async listGrantsRaw(requestParameters: ListGrantsRequest, initOverrides?: RequestInit): Promise<runtime.ApiResponse<Array<Grant>>> {
        const queryParameters: any = {};

        if (requestParameters.name !== undefined) {
            queryParameters['name'] = requestParameters.name;
        }

        if (requestParameters.kind !== undefined) {
            queryParameters['kind'] = requestParameters.kind;
        }

        if (requestParameters.destination !== undefined) {
            queryParameters['destination'] = requestParameters.destination;
        }

        const headerParameters: runtime.HTTPHeaders = {};

        if (this.configuration && this.configuration.accessToken) {
            const token = this.configuration.accessToken;
            const tokenString = await token("bearerAuth", []);

            if (tokenString) {
                headerParameters["Authorization"] = `Bearer ${tokenString}`;
            }
        }
        const response = await this.request({
            path: `/grants`,
            method: 'GET',
            headers: headerParameters,
            query: queryParameters,
        }, initOverrides);

        return new runtime.JSONApiResponse(response, (jsonValue) => jsonValue.map(GrantFromJSON));
    }

    /**
     * List grants
     */
    async listGrants(requestParameters: ListGrantsRequest, initOverrides?: RequestInit): Promise<Array<Grant>> {
        const response = await this.listGrantsRaw(requestParameters, initOverrides);
        return await response.value();
    }

}