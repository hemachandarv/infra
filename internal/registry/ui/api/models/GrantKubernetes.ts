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

import { exists, mapValues } from '../runtime';
import {
    GrantKubernetesKind,
    GrantKubernetesKindFromJSON,
    GrantKubernetesKindFromJSONTyped,
    GrantKubernetesKindToJSON,
} from './';

/**
 * 
 * @export
 * @interface GrantKubernetes
 */
export interface GrantKubernetes {
    /**
     * 
     * @type {GrantKubernetesKind}
     * @memberof GrantKubernetes
     */
    kind: GrantKubernetesKind;
    /**
     * 
     * @type {string}
     * @memberof GrantKubernetes
     */
    name: string;
    /**
     * 
     * @type {string}
     * @memberof GrantKubernetes
     */
    namespace: string;
}

export function GrantKubernetesFromJSON(json: any): GrantKubernetes {
    return GrantKubernetesFromJSONTyped(json, false);
}

export function GrantKubernetesFromJSONTyped(json: any, ignoreDiscriminator: boolean): GrantKubernetes {
    if ((json === undefined) || (json === null)) {
        return json;
    }
    return {
        
        'kind': GrantKubernetesKindFromJSON(json['kind']),
        'name': json['name'],
        'namespace': json['namespace'],
    };
}

export function GrantKubernetesToJSON(value?: GrantKubernetes | null): any {
    if (value === undefined) {
        return undefined;
    }
    if (value === null) {
        return null;
    }
    return {
        
        'kind': GrantKubernetesKindToJSON(value.kind),
        'name': value.name,
        'namespace': value.namespace,
    };
}

